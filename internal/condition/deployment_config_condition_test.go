package condition

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/stretchr/testify/assert"
)

func Test_non_user_creates_error_in_build_condition(t *testing.T) {
	user := "foo"
	condition := NewDCCondition(time.Duration(5) * time.Minute)
	_, err := condition.Eval(user)
	assert.Error(t, err, "Passing non User instances to Eval should return an error.")
}

func Test_eval_idle_for_deployment_config_condition_if_last_change_is_older_than_g_time(t *testing.T) {
	user := model.NewUser("123", "foo")
	user.JenkinsLastUpdate = time.Now().Add(-6 * time.Minute)
	condition := NewDCCondition(time.Duration(5) * time.Minute)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, Idle, result, "Condition should evaluate to Idle.")
}

func Test_eval_unidle_for_deployment_config_condition_if_last_change_is_younger_than_g_time(t *testing.T) {
	user := model.NewUser("123", "foo")
	user.JenkinsLastUpdate = time.Now().Add(-4 * time.Minute)
	condition := NewDCCondition(time.Duration(5) * time.Minute)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, UnIdle, result, "Condition should evaluate to UnIdle")
}
