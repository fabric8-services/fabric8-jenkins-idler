package toggles

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

// WARN these tests depend on a publicly available Unleash instance http://unleash.herokuapp.com
// These tests will fail if the service is down or the test toggle is being archived or changed.
// ATM, the following userId is configured to be valid 2e15e957-0366-4802-bf1e-0d6fe3f11bb6
// TODO - revisit this test and make it independent of the online version.
// TODO - See also https://github.com/fabric8-services/fabric8-jenkins-idler/issues/65

var features Features

func Test_idler_feature_disabled(t *testing.T) {
	setUp(t)
	enabled, err := features.IsIdlerEnabled("foo")
	if err != nil {
		assert.NoError(t, err)
	}

	assert.False(t, enabled, "The feature should not be enabled for 'foo'.")
}

func Test_idler_feature_enabled(t *testing.T) {
	setUp(t)
	enabled, err := features.IsIdlerEnabled("2e15e957-0366-4802-bf1e-0d6fe3f11bb6")
	if err != nil {
		assert.NoError(t, err)
	}

	assert.True(t, enabled, "The feature should not be enabled for '2e15e957-0366-4802-bf1e-0d6fe3f11bb6'.")
}

func setUp(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	var err error
	features, err = NewUnleashToggle("http://unleash.herokuapp.com/api/")
	if err != nil {
		assert.NoError(t, err)
	}
}
