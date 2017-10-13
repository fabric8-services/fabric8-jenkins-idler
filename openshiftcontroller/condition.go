package openshiftcontroller

import (
	"fmt"
	"time"
	"errors"

	log "github.com/sirupsen/logrus"
)

type ConditionI interface {
	IsTrueFor(object interface{}) (bool, error)
}

type Condition struct {
	ConditionI
}

type Conditions struct {
	Conditions map[string]ConditionI
}

func (c *Conditions) Eval(o interface{}) (result bool) {
	result = true
	for n, ci := range c.Conditions {
		//log.Info("Evaluating condition: ", n)
		r, err := ci.IsTrueFor(o)
		if err != nil {
			log.Error(err)
		} else if !r {
			log.Info("Condition ",n," is FALSE")
			result = false
		}
	}

	return result
}

type BuildCondition struct {
	Condition
	IdleAfter time.Duration
}

func NewBuildCondition(idleAfter time.Duration) *BuildCondition {
	b := &BuildCondition{IdleAfter: idleAfter}
	return b
}

func (c *BuildCondition) IsTrueFor(object interface{}) (result bool, err error) {
	result = false
	b, ok := object.(*User)
	if !ok {
		return false, errors.New(fmt.Sprintf("%s is not of type *User", object))
	}

	if !b.HasActive() && b.LastDone().Status.CompletionTimestamp.Time.Add(c.IdleAfter).Before(time.Now()) {
		result = true
	}

	return result, err

}
