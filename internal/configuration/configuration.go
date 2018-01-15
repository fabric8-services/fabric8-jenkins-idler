package configuration

import (
	"strings"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
	"os"
	"runtime"
	"strconv"
)

var (
	settings = map[string]Setting{}
)

func init() {
	settings["GetOpenShiftURL"] = Setting{"JC_OPENSHIFT_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetProxyURL"] = Setting{"JC_JENKINS_PROXY_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetTenantURL"] = Setting{"JC_F8TENANT_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetToggleURL"] = Setting{"JC_TOGGLE_API_URL", "", []func(interface{}, string) error{util.IsURL}}

	settings["GetOpenShiftToken"] = Setting{"JC_OPENSHIFT_API_TOKEN", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetAuthToken"] = Setting{"JC_AUTH_TOKEN", "", []func(interface{}, string) error{util.IsNotEmpty}}

	settings["GetIdleAfter"] = Setting{"JC_ILDE_AFTER", "30", []func(interface{}, string) error{util.IsInt}}
	settings["GetFilteredNamespaces"] = Setting{"JC_FILTER_NAMESPACES", "", []func(interface{}, string) error{}}
}

type Setting struct {
	key          string
	defaultValue string
	validations  []func(interface{}, string) error
}

type Configuration interface {
	// GetOpenShiftToken returns the OpenShift token.
	GetOpenShiftToken() string

	// GetOpenShiftURL returns the OpenShift API URL.
	GetOpenShiftURL() string

	// GetProxyURL returns the Jenkins Proxy API URL.
	GetProxyURL() string

	// GetTenantURL returns the F8 Tenant API URL.
	GetTenantURL() string

	// GetAuthToken returns the Auth token.
	GetAuthToken() string

	// GetToggleURL returns the Toggle Service URL.
	GetToggleURL() string

	// GetIdleAfter returns the number of minutes before Jenkins is idled.
	GetIdleAfter() int

	// GetFilteredNamespaces returns the list of namespaces to handle
	GetFilteredNamespaces() []string

	// Verify validates the configuration and returns an error in case the configuration is missing required settings
	// or contains invalid settings. If the configuration is correct nil is returned.
	Verify() util.MultiError
}

// EnvConfig reads the configuration from the environment
type EnvConfig struct {
}

// NewConfiguration creates a configuration instance
func NewConfiguration() (Configuration, error) {
	c := EnvConfig{}
	return &c, nil
}

// GetOpenShiftToken returns the OpenShift token as set via default, config file, or environment variable
func (c *EnvConfig) GetOpenShiftToken() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetOpenShiftURL returns the OpenShift API url as set via default, config file, or environment variable
func (c *EnvConfig) GetOpenShiftURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetProxyURL returns the Jenkins Proxy API URL as set via default, config file, or environment variable
func (c *EnvConfig) GetProxyURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetFilteredNamespaces returns the list of namespaces to handle as set via default, config file, or environment variable
func (c *EnvConfig) GetFilteredNamespaces() []string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	nameSpaces := strings.Split(value, ":")
	if len(nameSpaces) == 1 && len(nameSpaces[0]) == 0 {
		return []string{}
	}

	return nameSpaces
}

// GetIdleAfter returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable
func (c *EnvConfig) GetIdleAfter() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetTenantURL returns the F8 Tenant API URL as set via default, config file, or environment variable
func (c *EnvConfig) GetTenantURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetAuthToken returns the Auth token as set via default, config file, or environment variable
func (c *EnvConfig) GetAuthToken() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetToggleURL returns the Toggle Service URL as set via default, config file, or environment variable
func (c *EnvConfig) GetToggleURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

//Verify checks whether all needed config options are set
func (c *EnvConfig) Verify() util.MultiError {
	var errors util.MultiError
	for key, setting := range settings {
		value := c.getConfigValueFromEnv(key)

		for _, validateFunc := range setting.validations {
			errors.Collect(validateFunc(value, setting.key))
		}
	}

	return errors
}

func (c *EnvConfig) getConfigValueFromEnv(funcName string) string {
	setting := settings[funcName]

	value, ok := os.LookupEnv(setting.key)
	if !ok {
		value = setting.defaultValue
	}
	return value
}
