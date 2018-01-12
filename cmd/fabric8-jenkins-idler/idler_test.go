package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"sync"
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

	// register a global log hook to cpature the log output
	hook := test.NewGlobal()

	config, _ := configuration.NewData()
	idler := NewIdler(config, &mockFeatureToggle{})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		idler.Run()
	}()

	go func() {
		// Send SIGTERM after two seconds
		time.Sleep(2 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	wg.Wait()

	logMessages := extractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Received SIGTERM signal. Initiating shutdown.", "The recieval of the SIGTERM signal should have been looged")
	assert.Contains(t, logMessages, "Idler shutdown complete.", "Idler shutdown completion should have been logged")
}

func extractLogMessages(entries []*log.Entry) []string {
	messages := []string{}
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
