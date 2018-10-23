package condition

import (
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/sirupsen/logrus"
)

// DeploymentConfigCondition covers changes to DeploymentConfigs.
type DeploymentConfigCondition struct {
	idleAfter time.Duration
}

// NewDCCondition creates a new instance of DeploymentConfigCondition.
func NewDCCondition(idleAfter time.Duration) Condition {
	return &DeploymentConfigCondition{
		idleAfter: idleAfter,
	}
}

// Eval returns true if the last deployment config change occurred for more than the configured idle after interval.
func (c *DeploymentConfigCondition) Eval(object interface{}) (Action, error) {
	u, ok := object.(model.User)
	if !ok {
		return NoAction, fmt.Errorf("%T is not of type User", object)
	}

	log := logrus.WithFields(logrus.Fields{
		"id":        u.ID,
		"name":      u.Name,
		"component": "dc-condition",
	})

	lastUpdated := u.JenkinsLastUpdate
	if lastUpdated.IsZero() {
		log.WithField("action", "none").Info(
			"could not find when jenkins was last updated by idler, so taking no action")
		return NoAction, nil
	}

	now := time.Now().UTC()
	terminateTime := lastUpdated.Add(c.idleAfter)

	if now.After(terminateTime) {
		log.WithField("action", "idle").Infof("%v (%v) has elapsed after last update at %v",
			c.idleAfter, terminateTime, lastUpdated)
		return Idle, nil
	}

	log.WithField("action", "unidle").Infof(
		"%v (%v) has not elapsed after jenkins last update at %v",
		c.idleAfter, terminateTime, lastUpdated)
	return UnIdle, nil
}
