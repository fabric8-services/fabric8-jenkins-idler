package condition

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/stretchr/testify/assert"
)

func Test_non_user_creates_error(t *testing.T) {
	user := "foo"
	condition := NewBuildCondition(time.Duration(5)*time.Minute, time.Duration(5)*time.Minute)
	_, err := condition.Eval(user)
	assert.Error(t, err, "Passing non User instances to Eval should return an error.")
}

func Test_eval_idle_if_there_are_no_builds(t *testing.T) {
	user := model.NewUser("123", "foo")
	condition := NewBuildCondition(time.Duration(5)*time.Minute, time.Duration(5)*time.Minute)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, Idle, result, "Condition should evaluate to Idle.")
}

func Test_eval_unidle_when_active_build_exists(t *testing.T) {
	user := model.NewUser("123", "foo")
	user.ActiveBuild = model.Build{
		Metadata: model.Metadata{
			Name: "test build",
		},
		Status: model.Status{
			StartTimestamp: model.BuildTime{Time: time.Now()},
		},
	}
	condition := NewBuildCondition(time.Duration(5)*time.Minute, time.Duration(5)*time.Minute)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, UnIdle, result, "Condition should evaluate to UnIdle.")
}

func Test_eval_idle_when_activebuild_is_old(t *testing.T) {
	oldTime, _ := time.Parse(
		time.RFC3339,
		"1979-04-16T16:30:41+00:00")
	user := model.NewUser("123", "foo")
	user.ActiveBuild = model.Build{
		Metadata: model.Metadata{
			Name: "test build",
		},
		Status: model.Status{
			StartTimestamp: model.BuildTime{Time: oldTime},
		},
	}
	condition := NewBuildCondition(time.Duration(5)*time.Minute, time.Duration(10)*time.Hour)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, Idle, result, "Condition should evaluate to Idle.")
}

func Test_eval_completion_before_idletime_expires(t *testing.T) {
	user := model.NewUser("123", "foo")
	user.DoneBuild = model.Build{
		Metadata: model.Metadata{
			Name: "test build",
		},
		Status: model.Status{
			CompletionTimestamp: model.BuildTime{Time: time.Now()},
		},
	}
	condition := NewBuildCondition(time.Duration(5)*time.Minute, time.Duration(5)*time.Minute)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, NoAction, result, "Condition should evaluate to UnIdle.")
}

func Test_eval_completion_after_idletime_expires(t *testing.T) {
	oldTime, _ := time.Parse(
		time.RFC3339,
		"1979-04-16T16:30:41+00:00")

	user := model.NewUser("123", "foo")
	user.DoneBuild = model.Build{
		Metadata: model.Metadata{
			Name: "test build",
		},
		Status: model.Status{
			CompletionTimestamp: model.BuildTime{Time: oldTime},
		},
	}
	condition := NewBuildCondition(5*time.Minute, 5*time.Minute)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, Idle, result, "Condition should evaluate to Idle.")
}

func Test_eval_ignore_pesky_jenkins_sync_plugin(t *testing.T) {
	startTime := time.Now()
	completionTime := startTime.Add(time.Duration(4 * time.Second))
	user := model.NewUser("123", "foo")
	user.ActiveBuild = model.Build{
		Metadata: model.Metadata{
			Name: "test build",
		},
		Status: model.Status{
			Phase:               "New",
			StartTimestamp:      model.BuildTime{Time: startTime},
			CompletionTimestamp: model.BuildTime{Time: completionTime},
		},
	}
	condition := NewBuildCondition(time.Duration(5)*time.Minute, time.Duration(10)*time.Hour)
	result, err := condition.Eval(user)
	assert.NoError(t, err)
	assert.Equal(t, Idle, result, "Condition should evaluate to Idle.")
}
