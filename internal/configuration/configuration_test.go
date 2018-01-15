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
	JC_ILDE_AFTER            string
	JC_FILTER_NAMESPACES     []string
	errors                   []string
}

func Test_configuration_settings(t *testing.T) {

	var testConfigs = []config{
		{"http://localhost", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", []string{}, []string{}},
		{"https://localhost", "https://localhost", "https://localhost", "https://localhost", "token-1", "token-2", "10", []string{}, []string{}},
		{"", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", []string{}, []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"foo", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", []string{}, []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"/foo", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", []string{}, []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"ftp://snafu", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", []string{}, []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"", "http://localhost", "http://localhost", "http://localhost", "token-1", "token-2", "10", []string{}, []string{"Value for JC_OPENSHIFT_API_URL needs to be a valid URL."}},
		{"http://localhost", "", "http://localhost", "http://localhost", "token-1", "token-2", "10", []string{}, []string{"Value for JC_JENKINS_PROXY_API_URL needs to be a valid URL."}},
		{"http://localhost", "http://localhost", "", "http://localhost", "token-1", "token-2", "10", []string{}, []string{"Value for JC_F8TENANT_API_URL needs to be a valid URL."}},
		{"http://localhost", "http://localhost", "http://localhost", "", "token-1", "token-2", "10", []string{}, []string{"Value for JC_TOGGLE_API_URL needs to be a valid URL."}},
	}

	for _, testConfig := range testConfigs {
		for _, setting := range settings {
			os.Setenv(setting.key, getField(&testConfig, setting.key))
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

func getField(v *config, field string) string {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)
	return f.String()
}
