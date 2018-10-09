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
func (c *BuildCondition) Eval(object interface{}) (bool, error) {
	u, ok := object.(model.User)
	if !ok {
		return false, fmt.Errorf("%T is not of type User", object)
	}

	log := logrus.WithFields(logrus.Fields{
		"id":        u.ID,
		"name":      u.Name,
		"component": "build-condition",
	})

	if u.HasActiveBuilds() {
		// if we have activebuild being active over x time then see it as
		// expired or they would be lingering forever (i.e: approval process pipelines)
		startTime := u.ActiveBuild.Status.StartTimestamp.Time
		if startTime.Add(c.idleLongBuild).Before(time.Now()) {
			log.WithField("action", "idle").Infof(
				"active build started at %v has exceeded timeout %v",
				startTime, c.idleLongBuild)
			return true, nil
		}

		completionTime := u.ActiveBuild.Status.CompletionTimestamp.Time
		if u.ActiveBuild.Status.Phase == "New" &&
			startTime.Add(time.Second*5).After(completionTime) {
			log.WithField("action", "idle").Infof(
				"active build started at %v has gone past completion time %v",
				startTime, completionTime)
			return true, nil
		}

		log.WithField("action", "unidle").Infof(
			"active build started at %v seems to be in progress", startTime)
		return false, nil
	}

	if !u.HasBuilds() {
		log.WithField("action", "idle").Infof("user has no builds")
		return true, nil
	}

	completionTime := u.DoneBuild.Status.CompletionTimestamp.Time
	if completionTime.Add(c.idleAfter).Before(time.Now()) {
		log.WithField("action", "idle").Infof(
			"%v has elapsed after last done-build at %v ",
			c.idleAfter, completionTime)
		return true, nil
	}

	log.WithField("action", "unidle").Infof(
		"%v has not yet elapsed after last done-build at %v ",
		c.idleAfter, completionTime)
	return false, nil
}
