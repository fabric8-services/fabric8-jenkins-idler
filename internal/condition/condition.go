package condition

import (
	"fmt"
	"strings"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
	log "github.com/sirupsen/logrus"
)

// Condition defines a single Eval method which returns true or false.
type Condition interface {
	// Return true if the condition is true for a given object
	Eval(object interface{}) (bool, error)
}

type Conditions struct {
	Conditions map[string]Condition
}

// Eval evaluates a list of Conditions for a given object. It returns false if
// any of the conditions evaluates to false, otherwise true.
func (c *Conditions) Eval(o interface{}) (bool, util.MultiError) {
	result := true
	errors := util.MultiError{}

	condStates := make(map[string]bool)
	for name, ci := range c.Conditions {
		r, err := ci.Eval(o)
		if err != nil {
			log.Error(err)
			errors.Collect(err)
		}
		if !r {
			result = false
		}
		condStates[name] = r
	}

	log.Debugf("Conditions: %t = %s", result, c.conditionMapToString(condStates))
	return result, errors
}

func (c *Conditions) conditionMapToString(conditions map[string]bool) string {
	var result []string
	for key, value := range conditions {
		result = append(result, fmt.Sprintf("%s(%t)", key, value))
	}
	return strings.Join(result, " ")
}
