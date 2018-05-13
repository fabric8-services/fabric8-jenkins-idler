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
	userIdler := NewUserIdler(
		user, "", "", &mock.Config{},
		mock.NewMockFeatureToggle([]string{"42"}),
		&mock.TenantService{},
	)

	err := userIdler.checkIdle()
	assert.NoError(t, err, "No error expected.")

	logMessages := extractLogMessages(hook.Entries)
	assert.Contains(t, logMessages, "Idler not enabled.", "Conditions should have been evaluated.")
}

func Test_idle_check_returns_error_on_evaluation_failure(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	user := model.User{ID: "42"}
	userIdler := NewUserIdler(
		user, "", "", &mock.Config{},
		mock.NewMockFeatureToggle([]string{"42"}),
		&mock.TenantService{},
	)
	userIdler.Conditions.Add("error", &ErrorCondition{})

	err := userIdler.checkIdle()
	assert.Error(t, err, "Error expected.")
	assert.Equal(t, "eval error", err.Error(), "Unexpected error message.")
}

func Test_idle_check_occurs_even_without_openshift_events(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)
	hook := test.NewGlobal()

	user := model.User{ID: "100", Name: "John Doe"}
	config := &mock.Config{}
	features := mock.NewMockFeatureToggle([]string{"42"})
	tenantService := &mock.TenantService{}
	userIdler := NewUserIdler(user, "", "", config, features, tenantService)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(2300 * time.Millisecond)
		cancel()
	}()

	userIdler.Run(ctx, &wg, cancel, time.Duration(500*time.Millisecond), time.Duration(2000*time.Millisecond))

	userIdler.GetChannel() <- user
	time.Sleep(1100 * time.Millisecond)
	userIdler.GetChannel() <- user

	wg.Wait()

	logMessages := extractLogMessages(hook.Entries)
	idleAfterCount := 0
	userDataCount := 0
	resetCounterCounts := 0
	for _, message := range logMessages {
		if message == "Time based idle check." {
			idleAfterCount++
		}
		if message == "Received user data." {
			userDataCount++
		}
		if message == "Resetting retry counters." {
			resetCounterCounts++
		}
	}

	assert.Equal(t, 2, idleAfterCount, "Unexpected number of time based idle checks")
	assert.Equal(t, 2, userDataCount, "Unexpected number of user data events")
	assert.Equal(t, 1, resetCounterCounts, "Unexpected number of counter resets")

	assert.Contains(t, logMessages, "Shutting down user idler.", "No proper shutdown recorded.")
}

func Test_number_of_idle_calls_are_capped(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	hook := test.NewGlobal()

	user := model.User{ID: "42", Name: "John Doe"}
	openShiftClient := &mock.OpenShiftClient{}
	openShiftClient.IdleState = model.PodRunning

	config := &mock.Config{}
	maxRetry := 10
	config.MaxRetries = maxRetry
	features := mock.NewMockFeatureToggle([]string{"42"})
	tenantService := &mock.TenantService{}
	userIdler := NewUserIdler(user, "", "", config, features, tenantService)
	userIdler.openShiftClient = openShiftClient

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1000 * time.Millisecond)
		cancel()
	}()

	userIdler.Run(ctx, &wg, cancel, time.Duration(2000*time.Millisecond), time.Duration(2000*time.Millisecond))

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
	openShiftClient.IdleState = model.PodIdled

	config := &mock.Config{}
	maxRetry := 10
	config.MaxRetries = maxRetry

	features := mock.NewMockFeatureToggle([]string{"42"})
	tenantService := &mock.TenantService{}

	userIdler := NewUserIdler(user, "", "", config, features, tenantService)
	userIdler.openShiftClient = openShiftClient
	conditions := condition.NewConditions()
	conditions.Add("unidle", &UnIdleCondition{})
	userIdler.Conditions = &conditions

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1000 * time.Millisecond)
		cancel()
	}()

	userIdler.Run(ctx, &wg, cancel, time.Duration(2000*time.Millisecond), time.Duration(2000*time.Millisecond))

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
