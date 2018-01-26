package condition

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_non_user_creates_error_in_build_condition(t *testing.T) {
	user := "foo"
	condition := NewDCCondition(time.Duration(5) * time.Minute)
	_, err := condition.Eval(user)
	assert.Error(t, err, "Passing non User instances to Eval should return an error.")
}

func Test_eval_returns_true_for_deployment_config_condition_if_last_change_is_older_than_idle_time(t *testing.T) {
	user := model.NewUser("123", "foo", true)
	user.JenkinsLastUpdate = time.Now().Add(-6 * time.Minute)
	condition := NewDCCondition(time.Duration(5) * time.Minute)
	condValue, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.True(t, condValue, "Condition should evaluate to true.")
}

func Test_eval_returns_false_for_deployment_config_condition_if_last_change_is_younger_than_idle_time(t *testing.T) {
	user := model.NewUser("123", "foo", true)
	user.JenkinsLastUpdate = time.Now().Add(-4 * time.Minute)
	condition := NewDCCondition(time.Duration(5) * time.Minute)
	condValue, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.False(t, condValue, "Condition should evaluate to false.")
}
