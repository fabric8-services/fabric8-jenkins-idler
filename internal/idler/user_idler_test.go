package idler

import (
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
	return false, nil
}

func Test_Idle_Check_Skipped(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)

	// register a global log hook to capture the log output
	hook := test.NewGlobal()

	user := model.User{ID: "42"}
	userIdler := NewUserIdler(user, nil, &mock.MockConfig{}, &mockFeatureToggle{})

	userIdler.checkIdle()

	logMessages := extractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Idler not enabled.", "Conditions should have been evaluated.")
}

func extractLogMessages(entries []*log.Entry) []string {
	messages := []string{}
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
