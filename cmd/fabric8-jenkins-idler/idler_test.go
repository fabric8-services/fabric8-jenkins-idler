package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"syscall"
	"testing"
	"time"
)

type mockFeatureToggle struct {
}

func (m *mockFeatureToggle) IsIdlerEnabled(uid string) (bool, error) {
	return true, nil
}

func Test_graceful_shutdown(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	// register a global log hook to capture the log output
	hook := test.NewGlobal()

	config, _ := configuration.NewConfiguration()
	idler := NewIdler(config, &mockFeatureToggle{})

	go func() {
		// Send SIGTERM after two seconds
		time.Sleep(3 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	idler.Run()

	logMessages := extractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Idler successfully shut down.", "Idler shutdown completion should have been logged")
	assert.Contains(t, logMessages, "Stopping to watch OpenShift build configuration changes.", "Idler shutdown completion should have been logged")
	assert.Contains(t, logMessages, "Stopping to watch OpenShift deployment configuration changes.", "Idler shutdown completion should have been logged")
}

func extractLogMessages(entries []*log.Entry) []string {
	messages := []string{}
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
