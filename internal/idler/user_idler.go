package idler

import (
	"context"
	"sync"
	"time"

	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/condition"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{"component": "user-idler"})

const (
	bufferSize             = 10
	jenkinsServiceName     = "jenkins"
	jenkinsNamespaceSuffix = "-jenkins"
)

type UserIdler struct {
	openShiftClient client.OpenShiftClient
	maxRetries      int
	idleAttempts    int
	unIdleAttempts  int
	Conditions      *condition.Conditions
	logger          *log.Entry
	userChan        chan model.User
	user            model.User
	config          configuration.Configuration
	features        toggles.Features
}

func NewUserIdler(user model.User, openShiftClient client.OpenShiftClient, config configuration.Configuration, features toggles.Features) *UserIdler {
	logEntry := log.WithFields(log.Fields{
		"component": "user-idler",
		"username":  user.Name,
		"id":        user.ID,
	})
	logEntry.Info("UserIdler created.")

	conditions := createWatchConditions(config.GetProxyURL(), config.GetIdleAfter(), logEntry)

	userChan := make(chan model.User, bufferSize)

	userIdler := UserIdler{
		openShiftClient: openShiftClient,
		maxRetries:      config.GetMaxRetries(),
		idleAttempts:    0,
		unIdleAttempts:  0,
		Conditions:      conditions,
		logger:          logEntry,
		userChan:        userChan,
		user:            user,
		config:          config,
		features:        features,
	}
	return &userIdler
}

func (idler *UserIdler) GetChannel() chan model.User {
	return idler.userChan
}

// checkIdle verifies the state of conditions and decides if we should idle/unidle
// and performs the required action if needed
func (idler *UserIdler) checkIdle() error {
	eval, errors := idler.Conditions.Eval(idler.user)
	if !errors.Empty() {
		return errors.ToError()
	}

	idler.logger.WithField("eval", eval).Debug("Check idle state")
	if eval {
		enabled, err := idler.isIdlerEnabled()
		if err != nil {
			return err
		}
		if enabled {
			idler.doIdle()
		}
	} else {
		idler.doUnIdle()
	}

	return nil
}

func (idler *UserIdler) Run(wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc, checkIdle time.Duration) {
	idler.logger.Info("UserIdler started.")
	wg.Add(1)
	go func() {
		ticker := time.Tick(checkIdle)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				idler.logger.Info("Shutting down user idler.")
				cancel()
				return
			case idler.user = <-idler.userChan:
				idler.logger.WithField("data", idler.user.String()).Debug("Received user data.")

				err := idler.checkIdle()
				if err != nil {
					idler.logger.WithField("error", err.Error()).Warn("Error during idle check.")
				}
			case <-ticker:
				idler.logger.Info("Time based idle check.")
				idler.resetCounters()
				err := idler.checkIdle()
				if err != nil {
					idler.logger.WithField("error", err.Error()).Warn("Error during idle check.")
				}
			}
		}
	}()
}

func (idler *UserIdler) doIdle() error {
	if idler.idleAttempts >= idler.maxRetries {
		idler.logger.Warn("Skipping idle request since max retry count has been reached.")
		return nil
	}

	state, err := idler.getJenkinsState()
	if err != nil {
		return err
	}

	if state > model.JenkinsIdled {
		idler.incrementIdleAttempts()
		idler.logger.WithField("attempt", fmt.Sprintf("(%d/%d)", idler.idleAttempts, idler.maxRetries)).Info("About to idle Jenkins")
		err := idler.openShiftClient.Idle(idler.user.Name+jenkinsNamespaceSuffix, jenkinsServiceName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (idler *UserIdler) doUnIdle() error {
	if idler.unIdleAttempts >= idler.maxRetries {
		idler.logger.Warn("Skipping un-idle request since max retry count has been reached.")
		return nil
	}

	state, err := idler.getJenkinsState()
	if err != nil {
		return err
	}

	if state == model.JenkinsIdled {
		idler.incrementUnIdleAttempts()
		idler.logger.WithField("attempt", fmt.Sprintf("(%d/%d)", idler.unIdleAttempts, idler.maxRetries)).Info("About to un-idle Jenkins")
		err := idler.openShiftClient.UnIdle(idler.user.Name+jenkinsNamespaceSuffix, jenkinsServiceName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (idler *UserIdler) isIdlerEnabled() (bool, error) {
	enabled, err := idler.features.IsIdlerEnabled(idler.user.ID)
	if err != nil {
		return false, err
	}

	if enabled {
		logger.WithFields(log.Fields{"user": idler.user.Name, "uuid": idler.user.ID}).Debug("Idler enabled.")
		return true, nil
	} else {
		logger.WithFields(log.Fields{"user": idler.user.Name, "uuid": idler.user.ID}).Debug("Idler not enabled.")
		return false, nil
	}
}

func (idler *UserIdler) getJenkinsState() (int, error) {
	ns := idler.user.Name + jenkinsNamespaceSuffix
	state, err := idler.openShiftClient.IsIdle(ns, jenkinsServiceName)
	if err != nil {
		return -1, err
	}
	return state, nil
}

func (idler *UserIdler) incrementIdleAttempts() {
	idler.idleAttempts++
}

func (idler *UserIdler) incrementUnIdleAttempts() {
	idler.unIdleAttempts++
}

func (idler *UserIdler) resetCounters() {
	idler.idleAttempts = 0
	idler.unIdleAttempts = 0
}

func createWatchConditions(proxyUrl string, idleAfter int, logEntry *log.Entry) *condition.Conditions {
	conditions := condition.NewConditions()

	// Add a Build condition
	conditions.Add("build", condition.NewBuildCondition(time.Duration(idleAfter)*time.Minute))

	// Add a DeploymentConfig condition
	conditions.Add("DC", condition.NewDCCondition(time.Duration(idleAfter)*time.Minute))

	// If we have access to Proxy, add User condition
	if len(proxyUrl) > 0 {
		logEntry.Debug("Adding 'user' condition")
		conditions.Add("user", condition.NewUserCondition(proxyUrl, time.Duration(idleAfter)*time.Minute))
	}

	return &conditions
}
