package idler

import (
	"context"
	"errors"
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/condition"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

var logger = log.WithFields(log.Fields{"component": "user-idler"})

const (
	bufferSize = 10
)

type UserIdler struct {
	openShiftClient  client.OpenShiftClient
	maxUnIdleRetries int
	Conditions       *condition.Conditions
	logger           *log.Entry
	userChan         chan model.User
	user             model.User
	config           configuration.Configuration
	features         toggles.Features
}

func NewUserIdler(user model.User, openShiftClient client.OpenShiftClient, config configuration.Configuration, features toggles.Features) *UserIdler {
	logEntry := log.WithFields(log.Fields{
		"component": "user-idler",
		"username":  user.Name,
		"id":        user.ID,
	})

	conditions := createWatchConditions(config.GetProxyURL(), config.GetIdleAfter(), logEntry)

	userChan := make(chan model.User, bufferSize)

	userIdler := UserIdler{
		openShiftClient:  openShiftClient,
		maxUnIdleRetries: config.GetUnIdleRetry(),
		Conditions:       conditions,
		logger:           logEntry,
		userChan:         userChan,
		user:             user,
		config:           config,
		features:         features,
	}
	return &userIdler
}

func (idler *UserIdler) GetChannel() chan model.User {
	return idler.userChan
}

// checkIdle verifies the state of conditions and decides if we should idle/unidle
// and performs the required action if needed
func (idler *UserIdler) checkIdle() error {
	enabled, err := idler.features.IsIdlerEnabled(idler.user.ID)
	if err != nil {
		return err
	}

	if enabled {
		logger.WithFields(log.Fields{"user": idler.user.Name, "uuid": idler.user.ID}).Debug("Idler enabled. Evaluating conditions.")
	} else {
		logger.WithFields(log.Fields{"user": idler.user.Name, "uuid": idler.user.ID}).Debug("Idler not enabled. Skipping idle check.")
		return nil
	}

	eval := idler.Conditions.Eval(idler.user)

	if eval {
		idler.logger.WithField("eval", eval).Debug("Check idle state")
		idler.doIdle()
	} else {
		idler.logger.WithField("eval", eval).Debug("Check idle state")
		idler.doUnIdle()
	}

	return nil
}

func (idler *UserIdler) Run(wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				idler.logger.Info("Shutting down user idler.")
				cancel()
				return
			case idler.user = <-idler.userChan:
				idler.logger.Info("Receiving user data.")
				idler.checkIdle()
			case <-time.After(time.Duration(idler.config.GetIdleAfter()) * time.Minute):
				idler.logger.Info("IdleAfter timeout. Checking idle state.")
				idler.checkIdle()
			}
		}
	}()
}

func (idler *UserIdler) doIdle() error {
	//Check if Jenkins is running
	ns := idler.user.Name + "-jenkins"
	state, err := idler.openShiftClient.IsIdle(ns, "jenkins")
	if err != nil {
		return err
	}
	if state > model.JenkinsIdled {
		var n string
		var t time.Time
		if idler.user.HasCompletedBuilds() {
			n = idler.user.DoneBuild.Metadata.Name
			t = idler.user.DoneBuild.Status.CompletionTimestamp.Time
		}
		log.Info(fmt.Sprintf("I'd like to idle jenkins for %s as last build finished at %s", idler.user.Name, t))
		// Reset unidle retries and idle
		idler.user.UnIdleRetried = 0
		err := idler.openShiftClient.Idle(idler.user.Name+"-jenkins", "jenkins")
		if err != nil {
			return err
		}

		idler.user.AddJenkinsState(false, time.Now().UTC(), fmt.Sprintf("Jenkins Idled for %s, finished at %s", n, t))
	}
	return nil
}

func (idler *UserIdler) doUnIdle() error {
	ns := idler.user.Name + "-jenkins"
	state, err := idler.openShiftClient.IsIdle(ns, "jenkins")
	if err != nil {
		return err
	}
	if state == model.JenkinsIdled {
		log.Debug("Potential un-idling event")

		//Skip some retries,but check from time to time if things are fixed
		if idler.user.UnIdleRetried > idler.maxUnIdleRetries && (idler.user.UnIdleRetried%idler.maxUnIdleRetries != 0) {
			idler.user.UnIdleRetried++
			log.Debug(fmt.Sprintf("Skipping unidle for %s, too many retries", idler.user.Name))
			return nil
		}
		var n string
		var t time.Time
		if idler.user.HasActiveBuilds() {
			n = idler.user.ActiveBuild.Metadata.Name
			t = idler.user.ActiveBuild.Status.CompletionTimestamp.Time
		}
		//Inc unidle retries
		idler.user.UnIdleRetried++
		err := idler.openShiftClient.UnIdle(ns, "jenkins")
		if err != nil {
			return errors.New(fmt.Sprintf("Could not unidle Jenkins: %s", err))
		}
		idler.user.AddJenkinsState(true, time.Now().UTC(), fmt.Sprintf("Jenkins Unidled for %s at %s", n, t))
	}
	return nil
}

func createWatchConditions(proxyUrl string, idleAfter int, logEntry *log.Entry) *condition.Conditions {
	conditionsMap := make(map[string]condition.Condition)

	// Add a Build condition
	conditionsMap["build"] = condition.NewBuildCondition(time.Duration(idleAfter) * time.Minute)

	// Add a DeploymentConfig condition
	conditionsMap["DC"] = condition.NewDCCondition(time.Duration(idleAfter) * time.Minute)

	// If we have access to Proxy, add User condition
	if len(proxyUrl) > 0 {
		logEntry.Debug("Adding 'user' condition")
		conditionsMap["user"] = condition.NewUserCondition(proxyUrl, time.Duration(idleAfter)*time.Minute)
	}

	conditions := condition.Conditions{
		Conditions: conditionsMap,
	}

	return &conditions
}
