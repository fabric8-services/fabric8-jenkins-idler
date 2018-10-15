package openshift

import (
	"fmt"
	"time"

	"context"
	"sync"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	"github.com/sirupsen/logrus"
)

const (
	availableCond          = "Available"
	channelSendTimeout     = 1
	jenkinsNamespaceSuffix = "-jenkins"
)

var logger = logrus.WithFields(logrus.Fields{"component": "controller"})

// Controller defines the interface for watching the openShift cluster for changes.
type Controller interface {
	HandleBuild(o model.Object) error
}

// controllerImpl watches a single OpenShift cluster for Build changes.
// This struct needs to be safe for concurrent use.
type controllerImpl struct {
	openshiftURL  string
	osBearerToken string
	userIdlers    *UserIdlerMap
	tenantService tenant.Service
	features      toggles.Features
	config        configuration.Configuration
	wg            *sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	unknownUsers  *UnknownUsersMap
}

// NewController creates an instance of controllerImpl.
func NewController(
	ctx context.Context,
	openshiftURL string, osBearerToken string,
	userIdlers *UserIdlerMap,
	t tenant.Service,
	features toggles.Features,
	config configuration.Configuration,
	wg *sync.WaitGroup, cancel context.CancelFunc) Controller {

	logger.WithField("cluster", openshiftURL).Info("Creating new controller instance")

	controller := controllerImpl{
		openshiftURL:  openshiftURL,
		osBearerToken: osBearerToken,
		userIdlers:    userIdlers,
		tenantService: t,
		features:      features,
		config:        config,
		wg:            wg,
		ctx:           ctx,
		cancel:        cancel,
		unknownUsers:  NewUnknownUsersMap(),
	}

	return &controller
}

// HandleBuild processes new Build event collected from openShift and updates
// user structure with latest build info. NOTE: In most cases the only change in
// build object is stage timestamp, which we don't care about, so this function
// just does couple comparisons and returns.
func (c *controllerImpl) HandleBuild(o model.Object) error {
	ns := o.Object.Metadata.Namespace

	log := logger.WithFields(logrus.Fields{
		"ns":        ns,
		"event":     "build",
		"openshift": c.openshiftURL,
	})

	ok, err := c.createIfNotExist(ns)
	if err != nil {
		log.Errorf("Creating user-idler record failed: %s", err)
		return err
	}

	if !ok {
		return nil
	}

	userIdler := c.userIdlerForNamespace(ns)
	user := userIdler.GetUser()

	log = log.WithFields(logrus.Fields{
		"id":   user.ID,
		"name": user.Name,
	})

	evalConditions := false

	if c.isActive(&o.Object) {

		lastActive := user.ActiveBuild
		if lastActive.Status.Phase != o.Object.Status.Phase ||
			lastActive.Metadata.Name != o.Object.Metadata.Name {

			user.ActiveBuild = o.Object
			evalConditions = true
			log.Infof("Will send user %q to Idler due to active build", user.Name)
		}
	} else {

		lastDone := user.DoneBuild
		if lastDone.Status.Phase != o.Object.Status.Phase ||
			lastDone.Metadata.Name != o.Object.Metadata.Name {

			user.DoneBuild = o.Object
			evalConditions = true
			log.Infof("Will send user %q to Idler due to done build", user.Name)
		}
	}

	// If we have same build name (space name + build number) in Active and Done
	// it means last event was transition of an Active build into Done build
	// So we need to clean up the Active build ref.
	if user.ActiveBuild.Metadata.Name == user.DoneBuild.Metadata.Name {
		log.Infof("Active and Done builds are the same (%s), cleaning active builds", user.ActiveBuild.Metadata.Name)
		user.ActiveBuild = model.Build{Status: model.Status{Phase: "New"}}
		evalConditions = true
		log.Infof("Will send user %q to Idler due to transition of active build to a done build", user.Name)
	}

	if evalConditions {
		log.Infof("Sending user %q to Idler from a Build event", user.Name)
		sendUserToIdler(userIdler, user)
	}

	return nil
}

// createIfNotExist checks existence of a user in the map, initialise if it does not exist.
func (c *controllerImpl) createIfNotExist(ns string) (bool, error) {

	log := logger.WithField("ns", ns)
	if _, exist := c.unknownUsers.Load(ns); exist {
		log.Debugf("namespace %s listed in unknown users list", ns)
		return false, nil
	}

	if _, exist := c.userIdlers.Load(ns); exist {
		log.Debug("User idler found in cache")
		return true, nil
	}

	log.Infof("creating user-idler for cluster %s", c.openshiftURL)

	ti, err := c.tenantService.GetTenantInfoByNamespace(c.openshiftURL, ns)
	if err != nil {
		return false, err
	}

	if ti.Meta.TotalCount > 1 {
		return false, fmt.Errorf("could not add new user - Tenant service returned multiple items: %d", ti.Meta.TotalCount)
	} else if len(ti.Data) == 0 {

		// We can have a situation where a single OpenShift cluster is used by prod
		// as well as prod-preview and tenant service we connect to would only have
		// details about prod or prod-preview but we see events from tenants of both
		// prod and prod-preview which means in some cases we won't get tenant information
		// See: https://github.com/fabric8-services/fabric8-jenkins-idler/issues/155

		log.Debugf("No user info found for namespace %q", ns)
		log.Debugf("adding namespace: %s to unknown users list namespace", ns)
		c.unknownUsers.Store(ns, nil)
		return false, nil
	}

	userID := model.NewUser(ti.Data[0].ID, ns)
	userIdler := idler.NewUserIdler(
		userID, c.openshiftURL, c.osBearerToken,
		c.config, c.features, c.tenantService)
	c.userIdlers.Store(ns, userIdler)

	userIdler.Run(c.ctx, c.wg, c.cancel,
		time.Duration(c.config.GetCheckInterval())*time.Minute,
		time.Duration(c.config.GetMaxRetriesQuietInterval())*time.Minute)
	return true, nil
}

func (c *controllerImpl) userIdlerForNamespace(namespace string) *idler.UserIdler {
	idler, _ := c.userIdlers.Load(namespace)
	return idler
}

// isActive returns true if build phase suggests a build is active, false otherwise.
func (c *controllerImpl) isActive(b *model.Build) bool {
	return model.Phases[b.Status.Phase] == 1
}

func sendUserToIdler(idler *idler.UserIdler, user model.User) {
	select {
	case idler.GetChannel() <- user:
	case <-time.After(channelSendTimeout * time.Second):
		logger.WithField("ns", user.Name).Warn(
			"Unable to send user to channel. Discarding event.")
	}
}
