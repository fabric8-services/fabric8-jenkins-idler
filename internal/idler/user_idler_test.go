package idler

import (
	"context"
	"errors"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/condition"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

type ErrorCondition struct {
}

func (c *ErrorCondition) Eval(object interface{}) (bool, error) {
	return false, errors.New("eval error")
}

type UnIdleCondition struct {
}

func (c *UnIdleCondition) Eval(object interface{}) (bool, error) {
	return false, nil
}

func Test_idle_check_skipped_if_feature_not_enabled(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)

	// register a global log hook to capture the log output
	hook := test.NewGlobal()

	user := model.User{ID: "100"}
	userIdler := NewUserIdler(user, nil, &mock.Config{}, mock.NewMockFeatureToggle([]string{"42"}))

	err := userIdler.checkIdle()
	assert.NoError(t, err, "No error expected.")

	logMessages := extractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Idler not enabled.", "Conditions should have been evaluated.")
}

func Test_idle_check_returns_error_on_evaluation_failure(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	user := model.User{ID: "42"}
	userIdler := NewUserIdler(user, nil, &mock.Config{}, mock.NewMockFeatureToggle([]string{"42"}))
	userIdler.Conditions.Add("error", &ErrorCondition{})

	err := userIdler.checkIdle()
	assert.Error(t, err, "Error expected.")
	assert.Equal(t, "eval error", err.Error(), "Unexpected error message.")
}

func Test_timeout_occurs_regardless_of_other_events(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	hook := test.NewGlobal()

	user := model.User{ID: "100", Name: "John Doe"}
	openShiftClient := &mock.OpenShiftClient{}
	config := &mock.Config{}
	features := mock.NewMockFeatureToggle([]string{"42"})
	userIdler := NewUserIdler(user, openShiftClient, config, features)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(3500 * time.Millisecond)
		cancel()
	}()

	userIdler.Run(ctx, &wg, cancel, time.Duration(1*time.Second))

	userIdler.GetChannel() <- user
	time.Sleep(1 * time.Second)
	userIdler.GetChannel() <- user

	wg.Wait()

	logMessages := extractLogMessages(hook.Entries)
	idleAfterCount := 0
	userDataCount := 0
	for _, message := range logMessages {
		if message == "Time based idle check." {
			idleAfterCount++
		}
		if message == "Received user data." {
			userDataCount++
		}
	}

	assert.Equal(t, 3, idleAfterCount, "The timeout should have occurred 3 times.")
	assert.Equal(t, 2, userDataCount, "User data should have been received twice")

	assert.Contains(t, logMessages, "Shutting down user idler.", "NNo proper shutdown recorded.")
}

func Test_number_of_idle_calls_are_capped(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	hook := test.NewGlobal()

	user := model.User{ID: "42", Name: "John Doe"}
	openShiftClient := &mock.OpenShiftClient{}
	openShiftClient.IdleState = model.JenkinsRunning

	config := &mock.Config{}
	maxRetry := 3
	config.MaxRetries = maxRetry
	features := mock.NewMockFeatureToggle([]string{"42"})
	userIdler := NewUserIdler(user, openShiftClient, config, features)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1000 * time.Millisecond)
		cancel()
	}()

	userIdler.Run(ctx, &wg, cancel, time.Duration(2000*time.Millisecond))

	sendDataCount := 5
	for i := 0; i < sendDataCount; i++ {
		userIdler.GetChannel() <- user
	}

	wg.Wait()

	logMessages := extractLogMessages(hook.Entries)
	userDataCount := 0
	for _, message := range logMessages {
		if message == "Received user data." {
			userDataCount++
		}
	}

	assert.Equal(t, sendDataCount, userDataCount, "Wrong number of received data log entries")
	assert.Equal(t, maxRetry, openShiftClient.IdleCallCount, "Wrong idle call count")
	assert.Equal(t, 0, openShiftClient.UnIdleCallCount, "There should be no un-idle calls.")
}

func Test_number_of_unidle_calls_are_capped(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	hook := test.NewGlobal()

	user := model.User{ID: "42", Name: "John Doe"}

	openShiftClient := &mock.OpenShiftClient{}
	openShiftClient.IdleState = model.JenkinsIdled

	config := &mock.Config{}
	maxRetry := 3
	config.MaxRetries = maxRetry

	features := mock.NewMockFeatureToggle([]string{"42"})

	userIdler := NewUserIdler(user, openShiftClient, config, features)
	conditions := condition.NewConditions()
	conditions.Add("unidle", &UnIdleCondition{})
	userIdler.Conditions = &conditions

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1000 * time.Millisecond)
		cancel()
	}()

	userIdler.Run(ctx, &wg, cancel, time.Duration(2000*time.Millisecond))

	sendDataCount := 5
	for i := 0; i < sendDataCount; i++ {
		userIdler.GetChannel() <- user
	}

	wg.Wait()

	logMessages := extractLogMessages(hook.Entries)
	userDataCount := 0
	for _, message := range logMessages {
		if message == "Received user data." {
			userDataCount++
		}
	}

	assert.Equal(t, sendDataCount, userDataCount, "Wrong number of received data log entries")
	assert.Equal(t, maxRetry, openShiftClient.UnIdleCallCount, "Wrong un-idle call count")
	assert.Equal(t, 0, openShiftClient.IdleCallCount, "There should be no idle calls.")
}

func extractLogMessages(entries []*log.Entry) []string {
	messages := []string{}
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
