package configuration

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"strings"
	"testing"
)

type config struct {
	JC_OPENSHIFT_API_URL     string
	JC_JENKINS_PROXY_API_URL string
	JC_F8TENANT_API_URL      string
	JC_TOGGLE_API_URL        string
	JC_OPENSHIFT_API_TOKEN   string
	JC_AUTH_TOKEN            string
	JC_IDLE_AFTER            string
	JC_UN_IDLE_RETRY         string
	errors                   []string
}

func Test_configuration_settings(t *testing.T) {
	defer os.Clearenv()
	var testConfigs = []config{
		{"http://localhost", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", "15", []string{}},
		{"https://localhost", "https://localhost", "https://localhost", "https://localhost", "token-1", "token-2", "10", "15", []string{}},
		{"", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", "15", []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"foo", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", "15", []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"/foo", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", "15", []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"ftp://snafu", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", "15", []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", "15", []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"http://localhost", "", "http://localhost", "http://localhost", "token-1", "token-2", "10", "15", []string{"Value for JC_JENKINS_PROXY_API_URL needs to be a valid URL."}},
		{"http://localhost", "http://localhost", "", "http://localhost", "token-1", "token-2", "10", "15", []string{"Value for JC_F8TENANT_API_URL needs to be a valid URL."}},
		{"http://localhost", "http://localhost", "http://localhost", "", "token-1", "token-2", "10", "15", []string{"Value for JC_TOGGLE_API_URL needs to be a valid URL."}},
	}

	for _, testConfig := range testConfigs {
		for _, setting := range settings {
			configValue, ok := getConfigValueForEnvKey(&testConfig, setting.key)
			if ok {
				os.Setenv(setting.key, configValue)
			}
		}

		config, err := NewConfiguration()
		assert.NoError(t, err, "Creating the configuration failed unexpectedly.")

		multiError := config.Verify()
		var errorMessages []string
		if multiError.Empty() {
			errorMessages = []string{}
		} else {
			errorMessages = strings.Split(multiError.ToError().Error(), "\n")
		}

		assert.Equal(t, testConfig.errors, errorMessages, fmt.Sprintf("Errors don't match for config %v", testConfig))
	}

}

func Test_default_values(t *testing.T) {
	config, err := NewConfiguration()
	assert.NoError(t, err, "Creating the configuration failed unexpectedly.")
	assert.False(t, config.GetDebugMode(), "The default value for profileing should be false.")
	assert.Equal(t, config.GetIdleAfter(), defaultIdleAfter, "Unexpected default value for idle after.")
	assert.Equal(t, config.GetUnIdleRetry(), defaultIUnIdleRetry, "Unexpected default value for number of unidle retries.")
}

func getConfigValueForEnvKey(v *config, key string) (string, bool) {
	r := reflect.ValueOf(v)
	value := reflect.Indirect(r).FieldByName(key)
	if value.IsValid() {
		return value.String(), true
	} else {
		return "", false
	}
}
