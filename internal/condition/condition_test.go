package condition

import (
	"io/ioutil"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type IdleCondition struct {
}

func (c *IdleCondition) Eval(object interface{}) (Action, error) {
	return Idle, nil
}

type UnIdleCondition struct {
}

func (c *UnIdleCondition) Eval(object interface{}) (Action, error) {
	return UnIdle, nil
}

type ErrorCondition struct {
	msg string
}

func NewErrorCondition(msg string) Condition {
	b := &ErrorCondition{msg: msg}
	return b
}

func (c *ErrorCondition) Eval(object interface{}) (Action, error) {
	return NoAction, errors.New(c.msg)
}

func Test_all_conditions_idle(t *testing.T) {
	conditions := NewConditions()
	conditions.Add("idle-1", &IdleCondition{})
	conditions.Add("idle-2", &IdleCondition{})
	conditions.Add("idle-3", &IdleCondition{})

	result, err := conditions.Eval(model.NewUser("id", "name"))

	assert.NoError(t, err.ToError(), "No error expected.")
	assert.Equal(t, Action(Idle), result, "Should evaluate to Idle.")
}

func Test_all_conditions_unidle(t *testing.T) {
	conditions := NewConditions()
	conditions.Add("unidle-1", &UnIdleCondition{})
	conditions.Add("unidle-2", &UnIdleCondition{})
	conditions.Add("unidle-3", &UnIdleCondition{})

	result, err := conditions.Eval(model.NewUser("id", "name"))

	assert.NoError(t, err.ToError(), "No error expected.")
	assert.Equal(t, Action(UnIdle), result, "Should evaluate to unidle")
}

func Test_mixed_conditions(t *testing.T) {
	conditions := NewConditions()
	conditions.Add("idle-1", &IdleCondition{})
	conditions.Add("unidle-1", &UnIdleCondition{})
	conditions.Add("idle-2", &IdleCondition{})

	result, err := conditions.Eval(model.NewUser("id", "name"))

	assert.NoError(t, err.ToError(), "No error expected.")
	assert.Equal(t, Action(UnIdle), result, "Should evaluate to UnIdle.")
}

func Test_error_conditions(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	conditions := NewConditions()
	conditions.Add("idle-1", &IdleCondition{})
	conditions.Add("idle-2", &IdleCondition{})
	conditions.Add("error", NewErrorCondition("buh"))

	result, err := conditions.Eval(model.NewUser("id", "name"))

	assert.Error(t, err.ToError(), "No error expected.")
	assert.Equal(t, "buh", err.ToError().Error(), "Unexpected error message.")
	assert.Equal(t, Action(Idle), result, "Should evaluate to false.")
}
