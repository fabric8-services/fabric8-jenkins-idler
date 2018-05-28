package openshift

import (
	"fmt"
	"strconv"
	"time"

	"context"
	"sync"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	log "github.com/sirupsen/logrus"
)

const (
	availableCond          = "Available"
	channelSendTimeout     = 1
	jenkinsNamespaceSuffix = "-jenkins"
)

var logger = log.WithFields(log.Fields{"component": "controller"})

// Controller defines the interface for watching the openShift cluster for changes.
type Controller interface {
	HandleBuild(o model.Object) error
	HandleDeploymentConfig(dc model.DCObject) error
}

// controllerImpl watches a single OpenShift cluster for Build and Deployment Config changes. This struct needs to be
// safe for concurrent use.
type controllerImpl struct {
	openShiftAPIURL      string
	openShiftBearerToken string
	userIdlers           *UserIdlerMap
	openShiftClient      client.OpenShiftClient
	tenantService        tenant.Service
	features             toggles.Features
	config               configuration.Configuration
	wg                   *sync.WaitGroup
	ctx                  context.Context
	cancel               context.CancelFunc
	unknownUsers         *UnknownUsersMap
}

// NewController creates an instance of controllerImpl.
func NewController(ctx context.Context, openShiftAPI string, openShiftBearerToken string, userIdlers *UserIdlerMap, t tenant.Service, features toggles.Features, config configuration.Configuration, wg *sync.WaitGroup, cancel context.CancelFunc) Controller {
	logger.WithField("cluster", openShiftAPI).Info("Creating new controller instance")
	controller := controllerImpl{
		openShiftAPIURL:      openShiftAPI,
		openShiftBearerToken: openShiftBearerToken,
		userIdlers:           userIdlers,
		openShiftClient:      client.NewOpenShift(),
		tenantService:        t,
		features:             features,
		config:               config,
		wg:                   wg,
		ctx:                  ctx,
		cancel:               cancel,
		unknownUsers:         NewUnknownUsersMap(),
	}

	return &controller
}

// HandleBuild processes new Build event collected from openShift and updates
// user structure with latest build info. NOTE: In most cases the only change in
// build object is stage timestamp, which we don't care about, so this function
// just does couple comparisons and returns.
func (c *controllerImpl) HandleBuild(o model.Object) error {
	ns := o.Object.Metadata.Namespace
	ok, err := c.createIfNotExist(ns)
	if err != nil {
		return err
	}

	if !ok {
		// We can have a situation where a single OpenShift cluster is used by prod as well as prod-preview
		// For now we see in this case events from tenants of both clusters which means in some cases
		// we won't get tenant information
		// See also https://github.com/fabric8-services/fabric8-jenkins-idler/issues/155
		return nil
	}

	sendUserToIdler := false
	userIdler := c.userIdlerForNamespace(ns)
	user := userIdler.GetUser()
	if c.isActive(&o.Object) {
		lastActive := user.ActiveBuild
		if lastActive.Status.Phase != o.Object.Status.Phase || lastActive.Metadata.Name != o.Object.Metadata.Name {
			user.ActiveBuild = o.Object
			log.Infof("Will send user %v to idler due to active build", user.Name)
			sendUserToIdler = true
		}
	} else {
		lastDone := user.DoneBuild
		if lastDone.Status.Phase != o.Object.Status.Phase || lastDone.Metadata.Name != o.Object.Metadata.Name {
			user.DoneBuild = o.Object
			log.Infof("Will send user %v to idler due to done build", user.Name)
			sendUserToIdler = true
		}
	}

	// If we have same build name (space name + build number) in Active and Done build reference, it means last event was transition of an Active build into
	// Done build, we need to clean up the Active build ref.
	if user.ActiveBuild.Metadata.Name == user.DoneBuild.Metadata.Name {
		logger.WithFields(log.Fields{"ns": ns}).Infof("Active and Done builds are the same (%s), cleaning active builds", user.ActiveBuild.Metadata.Name)
		user.ActiveBuild = model.Build{Status: model.Status{Phase: "New"}}
		log.Infof("Will send user %v to idler due to transition of active build to a done build", user.Name)
		sendUserToIdler = true
	}

	if sendUserToIdler {
		log.Infof("Sending user %v to idler from a Build event", user.Name)
		c.sendUserToIdler(userIdler, user)
	}

	return nil
}

// HandleDeploymentConfig processes new DC event collected from openShift and updates
// user structure with info about the changes in DC. NOTE: This is important for cases
// like reset tenantService and update tenantService when DC is updated and Jenkins starts because
// of ConfigChange or manual intervention.
func (c *controllerImpl) HandleDeploymentConfig(dc model.DCObject) error {
	ns := dc.Object.Metadata.Namespace[:len(dc.Object.Metadata.Namespace)-len(jenkinsNamespaceSuffix)]
	ok, err := c.createIfNotExist(ns)
	if err != nil {
		return err
	}

	if !ok {
		// We can have a situation where a single OpenShift cluster is used by prod as well as prod-preview
		// For now we see in this case events from tenants of both clusters which means in some cases
		// we won't get tenant information
		// See also https://github.com/fabric8-services/fabric8-jenkins-idler/issues/155
		return nil
	}

	userIdler := c.userIdlerForNamespace(ns)
	user := userIdler.GetUser()
	sendUserToIdler := false

	condition, err := dc.Object.Status.GetByType(availableCond)
	if err != nil {
		return err
	}

	// TODO Verify if we need Generation vs. ObservedGeneration
	// This is either a new version of DC or we existing version waiting to come up.
	if (dc.Object.Metadata.Generation != dc.Object.Status.ObservedGeneration && dc.Object.Spec.Replicas > 0) || dc.Object.Status.UnavailableReplicas > 0 {
		user.JenkinsLastUpdate = time.Now().UTC()
		log.Infof("Will send user %v to idler due to a new version of DC or an existing version is coming up", user.Name)
		sendUserToIdler = true
	}

	// Also check if the event means that Jenkins just started (OS AvailableCondition.Status == true) and update time.
	status, err := strconv.ParseBool(condition.Status)
	if err != nil {
		return err
	}

	if status == true {
		user.JenkinsLastUpdate = condition.LastUpdateTime
		log.Infof("Will send user %v to idler because Jenkins was just started", user.Name)
		sendUserToIdler = true
	}

	if sendUserToIdler {
		log.Infof("Sending user %v to idler from a Deployment Config event", user.Name)
		c.sendUserToIdler(userIdler, user)
	}

	return nil
}

// createIfNotExist checks existence of a user in the map, initialise if it does not exist.
func (c *controllerImpl) createIfNotExist(ns string) (bool, error) {
	if _, exist := c.unknownUsers.Load(ns); exist {
		logger.WithField("ns", ns).Debug("Namespace listed in unknown users list")
		return false, nil
	}

	if _, exist := c.userIdlers.Load(ns); exist {
		logger.WithField("ns", ns).Debug("User idler found in cache")
		return true, nil
	}

	logger.WithField("ns", ns).Debug("Creating user idler")
	ti, err := c.tenantService.GetTenantInfoByNamespace(c.openShiftAPIURL, ns)
	if err != nil {
		return false, err
	}

	if ti.Meta.TotalCount > 1 {
		return false, fmt.Errorf("could not add new user - Tenant service returned multiple items: %d", ti.Meta.TotalCount)
	} else if len(ti.Data) == 0 {
		c.unknownUsers.Store(ns, nil)
		return false, nil
	}

	newUser := model.NewUser(ti.Data[0].ID, ns)
	userIdler := idler.NewUserIdler(newUser, c.openShiftAPIURL, c.openShiftBearerToken, c.config, c.features, c.tenantService)
	c.userIdlers.Store(ns, userIdler)
	userIdler.Run(c.ctx, c.wg, c.cancel, time.Duration(c.config.GetCheckInterval())*time.Minute, time.Duration(c.config.GetMaxRetriesQuietInterval())*time.Minute)
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

func (c *controllerImpl) sendUserToIdler(idler *idler.UserIdler, user model.User) {
	select {
	case idler.GetChannel() <- user:
	case <-time.After(channelSendTimeout * time.Second):
		logger.WithField("ns", user.Name).Warn("Unable to send user to channel. Discarding event.")
	}
}
