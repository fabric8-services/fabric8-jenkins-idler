package configuration

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validations(t *testing.T) {
	defer os.Clearenv()
	os.Clearenv()

	config, err := NewConfiguration()
	assert.NoError(t, err, "Creating the configuration failed unexpectedly.")

	multiError := config.Verify()
	var actualErrorMessages []string
	if multiError.Empty() {
		actualErrorMessages = []string{}
	} else {
		actualErrorMessages = strings.Split(multiError.ToError().Error(), "\n")
	}

	expectedErrors := []string{
		"value for JC_SERVICE_ACCOUNT_ID cannot be empty",
		"value for JC_SERVICE_ACCOUNT_SECRET cannot be empty",
		"value for JC_AUTH_TOKEN_KEY cannot be empty",
		"value for JC_F8TENANT_API_URL needs to be a valid URL",
		"value for JC_JENKINS_PROXY_API_URL needs to be a valid URL",
		"value for JC_F8TENANT_API_URL needs to be a valid URL",
		"value for JC_TOGGLE_API_URL needs to be a valid URL",
	}

	for _, verifyError := range expectedErrors {
		assert.Contains(t, actualErrorMessages, verifyError, "Expected error message is missing")
	}
}

func Test_default_values(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	config, err := NewConfiguration()
	assert.NoError(t, err, "Creating the configuration failed unexpectedly.")
	assert.False(t, config.GetDebugMode(), "The default value for profiling should be false.")
	assert.Equal(t, config.GetIdleAfter(), defaultIdleAfter, "Unexpected default value for idle after.")
	assert.Equal(t, config.GetIdleLongBuild(), defaultIdleLongBuild, "Unexpected default value for idle long build.")
	assert.Equal(t, config.GetMaxRetries(), defaultMaxRetries, "Unexpected default value for number of unidle retries.")
	assert.Equal(t, config.GetCheckInterval(), defaultCheckInterval, "Unexpected default value for number of unidle retries.")
}

func Test_fixed_uuid(t *testing.T) {
	defer os.Clearenv()

	var tests = []struct {
		value  string
		result []string
	}{
		{value: "42", result: []string{"42"}},
		{value: "42,1001", result: []string{"42", "1001"}},
		{value: "", result: []string{}},
	}

	for _, test := range tests {
		os.Clearenv()
		os.Setenv("JC_FIXED_UUIDS", test.value)

		config, err := NewConfiguration()
		assert.NoError(t, err, "Creating the configuration failed unexpectedly.")

		uuids := config.GetFixedUuids()
		assert.Equal(t, test.result, uuids, "Unexpected uuids.")
	}
}
