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
func (c *DeploymentConfigCondition) Eval(object interface{}) (bool, error) {
	u, ok := object.(model.User)
	if !ok {
		return false, fmt.Errorf("%T is not of type User", object)
	}

	log := logrus.WithFields(logrus.Fields{
		"id":        u.ID,
		"name":      u.Name,
		"component": "dc-condition",
	})

	if u.JenkinsLastUpdate.Add(c.idleAfter).Before(time.Now()) {
		log.WithField("action", "idle").Infof(
			"%v has elapsed after jenkins last update at %v",
			c.idleAfter, u.JenkinsLastUpdate)
		return true, nil
	}

	log.WithField("action", "unidle").Infof(
		"%v has not elapsed after jenkins last update at %v",
		c.idleAfter, u.JenkinsLastUpdate)

	return false, nil
}
