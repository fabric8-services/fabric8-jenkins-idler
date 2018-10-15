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
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithField("component", "user-idler")

// JenkinsServices is an array of all the services getting idled or unidled
// they go along the main build detection logic of jenkins and don't have
// any specific scenarios.
var JenkinsServices = []string{"jenkins"}

const (
	bufferSize             = 10
	jenkinsNamespaceSuffix = "-jenkins"
	jenkinsServiceName     = "jenkins"
)

// UserIdler is created for each monitored user/namespace.
// Each UserIdler runs in its own goroutine. The task of the UserIdler is to keep track
// of the Jenkins instance of the user and idle resp. un-idle depending on the evaluation
// of the given conditions for this UserIdler.
type UserIdler struct {
	openShiftAPI         string
	openShiftBearerToken string
	openShiftClient      client.OpenShiftClient
	maxRetries           int
	idleAttempts         int
	unIdleAttempts       int
	Conditions           *condition.Conditions
	logger               *log.Entry
	userChan             chan model.User
	user                 model.User
	config               configuration.Configuration
	features             toggles.Features
	tenantService        tenant.Service
}

// NewUserIdler creates an instance of UserIdler.
// It returns a pointer to UserIdler,
func NewUserIdler(
	user model.User,
	openShiftAPI, openShiftBearerToken string,
	config configuration.Configuration,
	features toggles.Features,
	tenantService tenant.Service) *UserIdler {

	logEntry := log.WithFields(log.Fields{
		"component": "user-idler",
		"name":      user.Name,
		"id":        user.ID,
	})
	logEntry.Info("UserIdler created.")

	conditions := createWatchConditions(config.GetProxyURL(), config.GetIdleAfter(), config.GetIdleLongBuild(), logEntry)

	userChan := make(chan model.User, bufferSize)

	userIdler := UserIdler{
		openShiftAPI:         openShiftAPI,
		openShiftBearerToken: openShiftBearerToken,
		openShiftClient:      client.NewOpenShift(),
		maxRetries:           config.GetMaxRetries(),
		idleAttempts:         0,
		unIdleAttempts:       0,
		Conditions:           conditions,
		logger:               logEntry,
		userChan:             userChan,
		user:                 user,
		config:               config,
		features:             features,
		tenantService:        tenantService,
	}
	return &userIdler
}

// GetUser returns the model.User of this idler.
func (idler *UserIdler) GetUser() model.User {
	return idler.user
}

// GetChannel gets channel of model.User type of this UserIdler.
func (idler *UserIdler) GetChannel() chan model.User {
	return idler.userChan
}

// checkIdle verifies the state of conditions and decides if we should idle/unidle
// and performs the required action if needed.
func (idler *UserIdler) checkIdle() error {

	enabled, err := idler.isIdlerEnabled()
	if err != nil {
		idler.logger.Errorf("Failed to check if idler is enabled for user: %s", err)
		return err
	}

	if !enabled {
		idler.logger.Warnf("idler disabled for user %s - skipping", idler.user.Name)
		return nil
	}

	idler.logger.Infof("Evaluating conditions for user %s", idler.user.Name)

	shouldIdle, errors := idler.Conditions.Eval(idler.user)
	if !errors.Empty() {
		idler.logger.Errorf("Failed to evaluate conditions for %s", idler.user.Name)
		return errors.ToError()
	}

	idler.logger.WithField("idle-jenkins", shouldIdle).Infof("Check idle state %s", idler.user.ID)

	if shouldIdle {
		err := idler.doIdle()
		// TODO: find a better way to update IdleStatus inside doIdle()
		idler.user.IdleStatus = model.NewIdleStatus(err)
	} else {
		err := idler.doUnIdle()
		// TODO: find a better way to update IdleStatus inside doUnIdle()
		idler.user.IdleStatus = model.NewUnidleStatus(err)
	}
	return nil
}

// Run runs/starts the Idler
// It checks if Jenkins is idle at every checkIdle duration.
func (idler *UserIdler) Run(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, checkIdle time.Duration, maxRetriesQuietInterval time.Duration) {
	idler.logger.WithFields(log.Fields{
		"checkIdle":               fmt.Sprintf("%.0fm", checkIdle.Minutes()),
		"maxRetriesQuietInterval": fmt.Sprintf("%.0fm", maxRetriesQuietInterval.Minutes()),
	}).Info("UserIdler started.")

	wg.Add(1)
	go func() {
		ticker := time.Tick(maxRetriesQuietInterval)
		timer := time.After(checkIdle)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				idler.logger.Info("Shutting down user idler.")
				cancel()
				return
			case idler.user = <-idler.userChan:
				idler.logger.WithField("state", idler.user.String()).Debug("Received user data.")

				err := idler.checkIdle()
				if err != nil {
					idler.logger.WithField("error", err.Error()).Warn("Error during idle check.")
				}
				// Resetting the timer
				timer = time.After(checkIdle)
			case <-timer:
				// Timer handles the case where there are no OpenShift events received for the user for the checkIdle
				// duration. This ensures checkIdle will be called regularly.
				idler.logger.WithField("state", idler.user.String()).Info("Time based idle check.")
				err := idler.checkIdle()
				if err != nil {
					idler.logger.WithField("error", err.Error()).Warn("Error during idle check.")
				}
			case <-ticker:
				// Using ticker for the resetting of counters to ensure it occurs
				idler.logger.Debug("Resetting retry counters.")
				idler.resetCounters()
			}
		}
	}()
}

func (idler *UserIdler) doIdle() error {
	if idler.idleAttempts >= idler.maxRetries {
		idler.logger.Warn("Skipping idle request since max retry count %d has reached.", idler.maxRetries)
		return nil
	}

	state, err := idler.getJenkinsState()
	if err != nil {
		return err
	}

	if state > model.PodIdled {
		idler.incrementIdleAttempts()
		for _, service := range JenkinsServices {

			// Let's add some more reasons, we probably want to
			reasonString := fmt.Sprintf("DoneBuild BuildName:%s Last:%s", idler.user.DoneBuild.Metadata.Name, idler.user.DoneBuild.Status.StartTimestamp.Time)
			if idler.user.ActiveBuild.Metadata.Name != "" {
				reasonString = fmt.Sprintf("ActiveBuild BuildName:%s Last:%s", idler.user.ActiveBuild.Metadata.Name, idler.user.ActiveBuild.Status.StartTimestamp.Time)
			}
			idler.logger.WithField("attempt", fmt.Sprintf("(%d/%d)", idler.idleAttempts, idler.maxRetries)).Info("About to idle " + service + ", Reason: " + reasonString)
			err := idler.openShiftClient.Idle(idler.openShiftAPI, idler.openShiftBearerToken, idler.user.Name+jenkinsNamespaceSuffix, service)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (idler *UserIdler) doUnIdle() error {

	idler.logger.Debugf("Current un-idle attempt count: %v, maximum retry count: %v", idler.unIdleAttempts, idler.maxRetries)
	if idler.unIdleAttempts >= idler.maxRetries {
		idler.logger.Warn("Skipping un-idle request since max retry count has been reached.")
		return nil
	}

	// The state can still return idled even though Jenkins is un-idled,
	// because we check for dc.status.replicas to determine if jenkins
	// is un-idled, which can still be 0 for some time after un-idling
	// TODO: measure the time taken for idler.getJenkinsState() to actually
	// change state from idled to un-idled, after a manual un-idling
	state, err := idler.getJenkinsState()
	if err != nil {
		return err
	}
	idler.logger.Infof("Current Jenkins' pod's state is %v", state)
	if state != model.PodIdled {
		return nil
	}

	ns := idler.user.Name + jenkinsNamespaceSuffix
	clusterFull, err := idler.tenantService.HasReachedMaxCapacity(idler.openShiftAPI, ns)
	if err != nil {
		return err
	} else if clusterFull {
		err := fmt.Errorf("Maximum Resource limit reached on %s for %s", idler.openShiftAPI, ns)
		return err
	}

	idler.incrementUnIdleAttempts()
	for _, service := range JenkinsServices {
		// Let's add some more reasons, we probably want to
		reasonString := fmt.Sprintf("DoneBuild BuildName:%s Last:%s", idler.user.DoneBuild.Metadata.Name, idler.user.DoneBuild.Status.StartTimestamp.Time)
		if idler.user.ActiveBuild.Metadata.Name != "" {
			reasonString = fmt.Sprintf("ActiveBuild BuildName:%s Last:%s", idler.user.ActiveBuild.Metadata.Name, idler.user.ActiveBuild.Status.StartTimestamp.Time)
		}
		idler.logger.WithField("attempt", fmt.Sprintf("(%d/%d)", idler.unIdleAttempts, idler.maxRetries)).Info("About to un-idle "+service+", Reason: ", reasonString)
		err := idler.openShiftClient.UnIdle(idler.openShiftAPI, idler.openShiftBearerToken, ns, service)
		if err != nil {
			idler.logger.Warnf("Failed to un-idle service %v in namespace %v (un-idle attempt: %v)", service, ns, idler.unIdleAttempts)
			idler.logger.Error(err)
			return err
		}
		idler.logger.Infof("Successfully un-idled service %v in namespace %v (un-idle attempt: %v)", service, ns, idler.unIdleAttempts)
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
	}

	logger.WithFields(log.Fields{"user": idler.user.Name, "uuid": idler.user.ID}).Debug("Idler not enabled.")
	return false, nil
}

func (idler *UserIdler) getJenkinsState() (model.PodState, error) {
	ns := idler.user.Name + jenkinsNamespaceSuffix
	state, err := idler.openShiftClient.State(idler.openShiftAPI, idler.openShiftBearerToken, ns, jenkinsServiceName)
	if err != nil {
		return model.PodStateUnknown, err
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

func createWatchConditions(proxyURL string, idleAfter int, idleLongBuild int, logEntry *log.Entry) *condition.Conditions {
	conditions := condition.NewConditions()

	// Add a Build condition.
	conditions.Add("build", condition.NewBuildCondition(
		time.Duration(idleAfter)*time.Minute,
		time.Duration(idleLongBuild)*time.Hour))

	return &conditions
}
