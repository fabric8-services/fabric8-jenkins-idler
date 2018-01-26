package condition

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_non_user_creates_error(t *testing.T) {
	user := "foo"
	condition := NewBuildCondition(time.Duration(5) * time.Minute)
	_, err := condition.Eval(user)
	assert.Error(t, err, "Passing non User instances to Eval should return an error.")
}

func Test_eval_returns_true_if_there_are_no_builds(t *testing.T) {
	user := model.NewUser("123", "foo", true)
	condition := NewBuildCondition(time.Duration(5) * time.Minute)
	condValue, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.True(t, condValue, "Condition should evaluate to true.")
}

func Test_eval_return_false_when_active_build_exists(t *testing.T) {
	user := model.NewUser("123", "foo", true)
	user.ActiveBuild = model.Build{
		Metadata: model.Metadata{
			Name: "test build",
		},
	}
	condition := NewBuildCondition(time.Duration(5) * time.Minute)
	condValue, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.False(t, condValue, "Condition should evaluate to false.")
}
