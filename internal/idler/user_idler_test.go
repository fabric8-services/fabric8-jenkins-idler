package idler

import (
	"errors"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

type mockFeatureToggle struct {
}

func (m *mockFeatureToggle) IsIdlerEnabled(uid string) (bool, error) {
	if uid == "42" {
		return true, nil
	}
	return false, nil
}

type ErrorCondition struct {
}

func (c *ErrorCondition) Eval(object interface{}) (bool, error) {
	return false, errors.New("eval error")
}

func Test_idle_check_skipped_if_feature_not_enabled(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)

	// register a global log hook to capture the log output
	hook := test.NewGlobal()

	user := model.User{ID: "100"}
	userIdler := NewUserIdler(user, nil, &mock.MockConfig{}, &mockFeatureToggle{})

	err := userIdler.checkIdle()
	assert.NoError(t, err, "No error expected.")

	logMessages := extractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Idler not enabled.", "Conditions should have been evaluated.")
}

func Test_idle_check_returns_error_on_evaluation_failure(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	user := model.User{ID: "42"}
	userIdler := NewUserIdler(user, nil, &mock.MockConfig{}, &mockFeatureToggle{})
	userIdler.Conditions.Add("error", &ErrorCondition{})

	err := userIdler.checkIdle()
	assert.Error(t, err, "Error expected.")
	assert.Equal(t, "eval error", err.Error(), "Unexpected error message.")
}

func extractLogMessages(entries []*log.Entry) []string {
	messages := []string{}
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
