package openshift

import (
	"fmt"
	"runtime"
	"strconv"
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
	HandleDeploymentConfig(dc model.DCObject) error
}

// controllerImpl watches a single OpenShift cluster for Build and Deployment Config changes. This struct needs to be
// safe for concurrent use.
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
			log.Infof("should evaluate conditions for %q due to active build", user.Name)
		}
	} else {

		lastDone := user.DoneBuild
		if lastDone.Status.Phase != o.Object.Status.Phase ||
			lastDone.Metadata.Name != o.Object.Metadata.Name {

			user.DoneBuild = o.Object
			evalConditions = true
			log.Infof("should evaluate conditions for %q due to completed build", user.Name)
		}
	}

	// If we have same build name (space name + build number) in Active and Done
	// it means last event was transition of an Active build into Done build
	// So we need to clean up the Active build ref.
	if user.ActiveBuild.Metadata.Name == user.DoneBuild.Metadata.Name {
		log.Infof("Active and Done builds are the same (%s), cleaning active builds", user.ActiveBuild.Metadata.Name)
		user.ActiveBuild = model.Build{Status: model.Status{Phase: "New"}}
		evalConditions = true
		log.Infof("should evaluate conditions for %q due to transition of active to  done build", user.Name)
	}

	if evalConditions {
		log.Infof("Sending user %q to user-idler for evaluating conditions", user.Name)
		sendUserToIdler(userIdler, user)
	}

	return nil
}

// HandleDeploymentConfig processes new DC event collected from openShift and updates
// user structure with info about the changes in DC. NOTE: This is important for cases
// like reset tenantService and update tenantService when DC is updated and Jenkins starts because
// of ConfigChange or manual intervention.
func (c *controllerImpl) HandleDeploymentConfig(dc model.DCObject) error {
	ns := dc.Object.Metadata.Namespace[:len(dc.Object.Metadata.Namespace)-len(jenkinsNamespaceSuffix)]

	log := logger.WithFields(logrus.Fields{
		"event":     "dc",
		"openshift": c.openshiftURL,
		"ns":        ns,
	})

	ok, err := c.createIfNotExist(ns)
	if err != nil {
		log.Errorf("Creating user-idler record failed: %s", err)
		return err
	}

	if !ok {
		return nil
	}

	// ensure user-idler is created for user so that pod would be
	// idled/unidled even if there aren't any build events
	userIdler := c.userIdlerForNamespace(ns)
	user := userIdler.GetUser()

	log = log.WithFields(logrus.Fields{
		"id":   user.ID,
		"name": user.Name,
	})

	availability, err := dc.Object.Status.GetByType(availableCond)
	if err != nil {
		// stop processing since the pod isn't available yet
		log.Errorf("available condition not present in the list of conditions - %s", err)
		return nil
	}

	// TODO(sthaha) Verify if we need Generation vs. ObservedGeneration
	// This is either a new version of DC or we existing version waiting to come up.
	// Log this so that we can use kibana logs to analyse if 'JenkinsLastUpdate'
	// should have been updated when this happens
	//
	if (dc.Object.Metadata.Generation != dc.Object.Status.ObservedGeneration && dc.Object.Spec.Replicas > 0) ||
		dc.Object.Status.UnavailableReplicas > 0 {
		log.Warnf("Noticed that a new version of jenkins has been deployed for %s but not setting lastupdate time", user.Name)
	}

	// Also check if the event means that Jenkins just started (OS AvailableCondition.Status == true) and update time.
	available, err := strconv.ParseBool(availability.Status)
	if err != nil {
		log.Errorf("could not parse availale condition status - %s", err)
		return err
	}

	if available {
		log.Infof("setting user jenkins-last-update to %v based on available condition", availability.LastUpdateTime)
		user.JenkinsLastUpdate = availability.LastUpdateTime
	}

	log.Infof("evaluate conditions for %q due to dc event", user.Name)
	sendUserToIdler(userIdler, user)
	return nil
}

// createIfNotExist checks existence of a user in the map, initialise if it does not exist.
func (c *controllerImpl) createIfNotExist(ns string) (bool, error) {

	log := logger.WithFields(logrus.Fields{
		"ns":        ns,
		"openshift": c.openshiftURL,
	})

	if _, exist := c.userIdlers.Load(ns); exist {
		log.Debug("User idler found in cache")
		return true, nil
	}

	if _, exist := c.unknownUsers.Load(ns); exist {
		log.Debugf("namespace %s listed in unknown users list", ns)
		return false, nil
	}

	log.Infof("creating user-idler for cluster %s", c.openshiftURL)

	ti, err := c.tenantService.GetTenantInfoByNamespace(c.openshiftURL, ns)
	if err != nil {
		return false, err
	}

	if ti.Meta.TotalCount > 1 {
		return false, fmt.Errorf("could not add new user - Tenant service returned multiple items: %d", ti.Meta.TotalCount)
	} else if len(ti.Data) == 0 {
		log.Warnf("adding namespace: %s to unknown users list namespace", ns)
		c.unknownUsers.Store(ns, nil)
		return false, nil
	}

	log.Warnf("tenant info from tenant-service %v", ti)
	user := model.NewUser(ti.Data[0].ID, ns)

	userIdler := idler.NewUserIdler(
		user, c.openshiftURL, c.osBearerToken,
		c.config, c.features, c.tenantService)

	c.userIdlers.Store(ns, userIdler)

	userIdler.Run(c.ctx, c.wg, c.cancel,
		time.Duration(c.config.GetCheckInterval())*time.Minute,
		time.Duration(c.config.GetMaxRetriesQuietInterval())*time.Minute)

	idlerCount := c.userIdlers.Len()
	goRoutines := runtime.NumGoroutine()

	log.WithFields(logrus.Fields{
		"user_idler.count": idlerCount,
		"go.routines":      goRoutines,
	}).Infof("created user-idler [%d] | cluster %s | ns: %s | gr: %d",
		idlerCount, c.openshiftURL, ns, goRoutines)
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
