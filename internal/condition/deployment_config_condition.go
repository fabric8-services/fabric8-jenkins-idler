package condition

import (
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
)

// DeploymentConfigCondition covers changes to Jenkins DeploymentConfigs
type DeploymentConfigCondition struct {
	idleAfter time.Duration
}

// NewDCCondition creates a new instance of DeploymentConfigCondition
func NewDCCondition(idleAfter time.Duration) Condition {
	b := &DeploymentConfigCondition{
		idleAfter: idleAfter,
	}
	return b
}

// Eval returns true if the last deployment config change occurred for more than the configured idle after interval.
func (c *DeploymentConfigCondition) Eval(object interface{}) (bool, error) {
	b, ok := object.(model.User)
	if !ok {
		return false, fmt.Errorf("%T is not of type User", object)
	}

	if b.JenkinsLastUpdate.Add(c.idleAfter).Before(time.Now()) {
		return true, nil
	}

	return false, nil
}
