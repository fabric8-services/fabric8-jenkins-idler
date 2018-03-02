package condition

import (
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
)

// BuildCondition covers builds a user has/had running.
type BuildCondition struct {
	idleAfter time.Duration
}

// NewBuildCondition creates a new instance of BuildCondition given
// idleAfter(time after which jenkins should be idled).
func NewBuildCondition(idleAfter time.Duration) Condition {
	b := &BuildCondition{idleAfter: idleAfter}
	return b
}

// Eval returns true if the passed User does not have any builds or does not have any
// active builds and the time elapsed since the last completed build is created than the configured idle after time.
func (c *BuildCondition) Eval(object interface{}) (bool, error) {
	u, ok := object.(model.User)
	if !ok {
		return false, fmt.Errorf("%T is not of type User", object)
	}

	if u.HasActiveBuilds() {
		return false, nil
	}

	if !u.HasBuilds() {
		return true, nil
	}

	if u.DoneBuild.Status.CompletionTimestamp.Time.Add(c.idleAfter).Before(time.Now()) {
		return true, nil
	}

	return false, nil
}
