package condition

import (
	"fmt"
	"strings"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
	"github.com/sirupsen/logrus"
)

// Action is a tri-state enum for  different actions to be applied to Pod.
// E.g. Pods can be Idled, UnIdled or Left at its current state.
type Action int

const (
	// NoAction  an unknown state of the Pod. Used usually with Error.
	NoAction Action = 0
	// Idle represents the idled state of a Pod.
	Idle = 1
	// UnIdle state is when Pods are about to start.
	UnIdle = 2

	// unknownAction used internally for range check
	unknownAction = 3
)

func (a Action) String() string {
	actions := [...]string{
		"no action",
		"idle",
		"unidle",
	}
	if a < NoAction || a >= unknownAction {
		return "unknown action"
	}
	return actions[a]
}

// Condition defines a single Eval method which returns true or false.
type Condition interface {
	// Return true if the condition is true for a given object.
	Eval(object interface{}) (Action, error)
}

// Conditions defines map of Condition instances by their names
type Conditions struct {
	conditions map[string]Condition
}

// NewConditions create a new instance of Conditions.
func NewConditions() Conditions {
	return Conditions{
		conditions: make(map[string]Condition),
	}
}

// Eval evaluates a list of Conditions for a given object. It returns false if
// any of the conditions evaluates to false, otherwise true.
func (c *Conditions) Eval(o interface{}) (Action, util.MultiError) {
	errors := util.MultiError{}

	u, ok := o.(model.User)
	if !ok {
		errors.Collect(fmt.Errorf("%T is not of type User", o))
		return NoAction, errors
	}

	log := logrus.WithFields(logrus.Fields{
		"id":        u.ID,
		"name":      u.Name,
		"component": "condition",
	})

	condStates := make(map[string]Action)

	result := NoAction
	for name, ci := range c.conditions {
		action, err := ci.Eval(o)

		if err != nil {
			log.Error(err)
			errors.Collect(err)
		}

		condStates[name] = action

		// overall result is the max of all conditions
		if action > result {
			result = action
		}
		// TODO(sthaha): skip rest of the condition check is any of the
		// condition results in UnIdle
	}

	log.Infof("conditions/result: %s | %s", result, c.conditionMapToString(condStates))
	return result, errors
}

// Add adds a condition with its name to the this Conditions instance.
func (c *Conditions) Add(name string, condition Condition) {
	c.conditions[name] = condition
}

func (c *Conditions) conditionMapToString(conditions map[string]Action) string {
	var result []string
	for key, value := range conditions {
		result = append(result, fmt.Sprintf("%s: %s", key, value))
	}
	return strings.Join(result, " | ")
}
