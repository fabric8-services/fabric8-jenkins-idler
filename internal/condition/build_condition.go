package condition

import (
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/sirupsen/logrus"
)

// BuildCondition covers builds a user has/had running.
type BuildCondition struct {
	idleAfter     time.Duration
	idleLongBuild time.Duration
}

// NewBuildCondition creates a new instance of BuildCondition given
// idleAfter(time after which jenkins should be idled).
func NewBuildCondition(idleAfter time.Duration, idleLongBuild time.Duration) Condition {
	b := &BuildCondition{idleAfter: idleAfter, idleLongBuild: idleLongBuild}
	return b
}

// Eval returns true if the passed User does not have any builds or does not have any
// active builds and the time elapsed since the last completed build is created than the configured idle after time.
func (c *BuildCondition) Eval(object interface{}) (Action, error) {
	u, ok := object.(model.User)
	if !ok {
		return NoAction, fmt.Errorf("%T is not of type User", object)
	}

	log := logrus.WithFields(logrus.Fields{
		"id":        u.ID,
		"name":      u.Name,
		"component": "build-condition",
	})

	log.WithField("check", "any-builds").Infof("Checking if there are any builds")
	if !u.HasBuilds() {
		log.WithField("action", "idle").Infof("user has no builds")
		return Idle, nil
	}

	now := time.Now().UTC()

	log.WithField("check", "active-builds").Infof("Checking active builds")
	if u.HasActiveBuilds() {
		// if we have activebuild being active over x time then see it as
		// expired or they would be lingering forever (i.e: approval process pipelines)

		startTime := u.ActiveBuild.Status.StartTimestamp.Time
		maxBuildTime := startTime.Add(c.idleLongBuild)
		log.WithField("check", "active-build-timedout").Infof(
			"has active build started at %v has exceeded max build time: %v ", startTime, maxBuildTime)

		if now.After(maxBuildTime) {
			log.WithField("action", "idle").Infof(
				"active build started at %v has exceeded timeout %v", startTime, c.idleLongBuild)
			return Idle, nil
		}

		completionTime := u.ActiveBuild.Status.CompletionTimestamp.Time

		log.WithField("check", "active-build-completed").Infof(
			"has active build started at %v gone past completion time %v", startTime, completionTime)
		if u.ActiveBuild.Status.Phase == "New" &&
			completionTime.Sub(startTime) > 2*time.Second {
			log.WithField("action", "idle").Infof(
				"active build started at %v has gone past completion time %v",
				startTime, completionTime)
			/// TODO: not sure about this
			return Idle, nil
		}

		log.WithField("action", "unidle").Infof(
			"active build started at %v seems to be in progress", startTime)
		return UnIdle, nil
	}

	// Done builds

	log.WithField("check", "done-builds").Infof("Checking done builds")

	if u.DoneBuild.Status.Phase == "Cancelled" {
		log.WithField("action", "idle").Infof("Build is cancelled")
		return Idle, nil
	}

	completionTime := u.DoneBuild.Status.CompletionTimestamp.Time
	terminateTime := completionTime.Add(c.idleAfter)

	log.WithField("check", "done-builds").Infof(
		"Check if completion time %v is past terminate time: %v", now, terminateTime)

	if now.After(terminateTime) {
		log.WithField("action", "idle").Infof(
			"%v has elapsed after last done-build at %v ", c.idleAfter, completionTime)
		return Idle, nil
	}

	log.WithField("action", "none").Infof(
		"%v has not yet elapsed after last done-build at %v ", c.idleAfter, completionTime)
	return NoAction, nil
}
