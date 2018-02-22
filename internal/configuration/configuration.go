package configuration

import (
	"strings"

	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
	"os"
	"runtime"
	"strconv"
)

const (
	defaultIdleAfter    = 45
	defaultIUnIdleRetry = 10
)

var (
	settings = map[string]Setting{}
)

func init() {
	// service URLs for required services
	settings["GetOpenShiftURL"] = Setting{"JC_OPENSHIFT_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetProxyURL"] = Setting{"JC_JENKINS_PROXY_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetTenantURL"] = Setting{"JC_F8TENANT_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetToggleURL"] = Setting{"JC_TOGGLE_API_URL", "", []func(interface{}, string) error{util.IsURL}}

	// required secrets
	settings["GetOpenShiftToken"] = Setting{"JC_OPENSHIFT_API_TOKEN", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetAuthToken"] = Setting{"JC_AUTH_TOKEN", "", []func(interface{}, string) error{util.IsNotEmpty}}

	// timeouts and retry counts
	settings["GetIdleAfter"] = Setting{"JC_IDLE_AFTER", strconv.Itoa(defaultIdleAfter), []func(interface{}, string) error{util.IsInt}}
	settings["GetUnIdleRetry"] = Setting{"JC_UN_IDLE_RETRY", strconv.Itoa(defaultIUnIdleRetry), []func(interface{}, string) error{util.IsInt}}

	// debug
	settings["GetDebugMode"] = Setting{"JC_DEBUG_MODE", "false", []func(interface{}, string) error{util.IsBool}}
	settings["GetFixedUuids"] = Setting{"JC_FIXED_UUIDS", "", []func(interface{}, string) error{}}
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

	// GetUnIdleRetry returns the maximum number of retries to un-idle the Jenkins service.
	GetUnIdleRetry() int

	// GetDebugMode returns if debug mode should be enabled.
	GetDebugMode() bool

	// GetFixedUuids returns a slice of fixed user uuids. If set, a custom Features implementation is instantiated
	// which only enabled the idler feature for the specified list of users. This is mainly used for local dev only.
	GetFixedUuids() []string

	// Verify validates the configuration and returns an error in case the configuration is missing required settings
	// or contains invalid settings. If the configuration is correct nil is returned.
	Verify() util.MultiError

	// String returns the current configuration as string.
	String() string
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

// GetIdleAfter returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable
func (c *EnvConfig) GetIdleAfter() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

//  GetUnIdleRetry returns the maximum number of retries to un-idle the Jenkins service
func (c *EnvConfig) GetUnIdleRetry() int {
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

// GetDebugMode returns if debug mode should be enabled.
func (c *EnvConfig) GetDebugMode() bool {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	b, _ := strconv.ParseBool(value)

	return b
}

// GetFixedUuids returns a slice of fixed user uuids. The uuids are specified comma separated in the environment variable
// JC_FIXED_UUIDS.
func (c *EnvConfig) GetFixedUuids() []string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	if len(value) == 0 {
		return []string{}
	}

	return strings.Split(value, ",")
}

// Verify checks whether all needed config options are set
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

func (c *EnvConfig) String() string {
	config := map[string]interface{}{}
	for key, setting := range settings {
		value := c.getConfigValueFromEnv(key)
		// don't echo tokens
		if strings.Contains(setting.key, "TOKEN") && len(value) > 0 {
			value = "***"
		}
		config[key] = value

	}
	return fmt.Sprintf("%v", config)
}

func (c *EnvConfig) getConfigValueFromEnv(funcName string) string {
	setting := settings[funcName]

	value, ok := os.LookupEnv(setting.key)
	if !ok {
		value = setting.defaultValue
	}
	return value
}
