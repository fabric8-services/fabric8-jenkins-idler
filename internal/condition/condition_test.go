package condition

import (
	"io/ioutil"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type TrueCondition struct {
}

func (c *TrueCondition) Eval(object interface{}) (bool, error) {
	return true, nil
}

type FalseCondition struct {
}

func (c *FalseCondition) Eval(object interface{}) (bool, error) {
	return false, nil
}

type ErrorCondition struct {
	msg string
}

func NewErrorCondition(msg string) Condition {
	b := &ErrorCondition{msg: msg}
	return b
}

func (c *ErrorCondition) Eval(object interface{}) (bool, error) {
	return false, errors.New(c.msg)
}

func Test_all_conditions_true(t *testing.T) {
	conditions := NewConditions()
	conditions.Add("true-1", &TrueCondition{})
	conditions.Add("true-2", &TrueCondition{})
	conditions.Add("true-3", &TrueCondition{})

	eval, err := conditions.Eval(model.NewUser("id", "name"))

	assert.NoError(t, err.ToError(), "No error expected.")
	assert.True(t, eval, "Should evaluate to true.")
}

func Test_all_conditions_false(t *testing.T) {
	conditions := NewConditions()
	conditions.Add("false-1", &FalseCondition{})
	conditions.Add("false-2", &FalseCondition{})
	conditions.Add("false-3", &FalseCondition{})

	eval, err := conditions.Eval(model.NewUser("id", "name"))

	assert.NoError(t, err.ToError(), "No error expected.")
	assert.False(t, eval, "Should evaluate to false.")
}

func Test_mixed_conditions(t *testing.T) {
	conditions := NewConditions()
	conditions.Add("true-1", &TrueCondition{})
	conditions.Add("true-2", &TrueCondition{})
	conditions.Add("false-1", &FalseCondition{})

	eval, err := conditions.Eval(model.NewUser("id", "name"))

	assert.NoError(t, err.ToError(), "No error expected.")
	assert.False(t, eval, "Should evaluate to false.")
}

func Test_error_conditions(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	conditions := NewConditions()
	conditions.Add("true-1", &TrueCondition{})
	conditions.Add("true-2", &TrueCondition{})
	conditions.Add("error", NewErrorCondition("buh"))

	eval, err := conditions.Eval(model.NewUser("id", "name"))

	assert.Error(t, err.ToError(), "No error expected.")
	assert.Equal(t, "buh", err.ToError().Error(), "Unexpected error message.")
	assert.False(t, eval, "Should evaluate to false.")
}
