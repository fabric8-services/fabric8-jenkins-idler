package configuration

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

const (
	defaultIdleLongBuild           = 3
	defaultIdleAfter               = 45
	defaultMaxRetries              = 10
	defaultMaxRetriesQuietInterval = 30
	defaultCheckInterval           = 15
)

var (
	settings = map[string]Setting{}
)

func init() {
	// service URLs for required services
	settings["GetProxyURL"] = Setting{"JC_JENKINS_PROXY_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetTenantURL"] = Setting{"JC_F8TENANT_API_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetToggleURL"] = Setting{"JC_TOGGLE_API_URL", "", []func(interface{}, string) error{util.IsURL}}

	// Auth service id and secret as well as key to decrypt OpenShift API tokens
	settings["GetAuthURL"] = Setting{"JC_AUTH_URL", "", []func(interface{}, string) error{util.IsURL}}
	settings["GetServiceAccountID"] = Setting{"JC_SERVICE_ACCOUNT_ID", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetServiceAccountSecret"] = Setting{"JC_SERVICE_ACCOUNT_SECRET", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetAuthTokenKey"] = Setting{"JC_AUTH_TOKEN_KEY", "", []func(interface{}, string) error{util.IsNotEmpty}}
	settings["GetAuthGrantType"] = Setting{"JC_AUTH_GRANT_TYPE", "client_credentials", []func(interface{}, string) error{util.IsNotEmpty}}

	// timeouts and retry counts
	settings["GetIdleAfter"] = Setting{"JC_IDLE_AFTER", strconv.Itoa(defaultIdleAfter), []func(interface{}, string) error{util.IsInt}}
	settings["GetIdleLongBuild"] = Setting{"JC_IDLE_LONG_BUILD", strconv.Itoa(defaultIdleLongBuild), []func(interface{}, string) error{util.IsInt}}
	settings["GetMaxRetries"] = Setting{"JC_MAX_RETRIES", strconv.Itoa(defaultMaxRetries), []func(interface{}, string) error{util.IsInt}}
	settings["GetMaxRetriesQuietInterval"] = Setting{"JC_MAX_RETRIES_QUIET_INTERVAL", strconv.Itoa(defaultMaxRetriesQuietInterval), []func(interface{}, string) error{util.IsInt}}
	settings["GetCheckInterval"] = Setting{"JC_CHECK_INTERVAL", strconv.Itoa(defaultCheckInterval), []func(interface{}, string) error{util.IsInt}}

	// debug
	settings["GetDebugMode"] = Setting{"JC_DEBUG_MODE", "false", []func(interface{}, string) error{util.IsBool}}
	settings["GetFixedUuids"] = Setting{"JC_FIXED_UUIDS", "", []func(interface{}, string) error{}}
}

// Setting defines an element in the configuration of Jenkins Idler.
type Setting struct {
	key          string
	defaultValue string
	validations  []func(interface{}, string) error
}

// EnvConfig reads the configuration from the environment.
type EnvConfig struct {
}

// NewConfiguration creates a configuration instance.
func NewConfiguration() (Configuration, error) {
	c := EnvConfig{}
	return &c, nil
}

// GetProxyURL returns the Jenkins Proxy API URL as set via default, config file, or environment variable.
func (c *EnvConfig) GetProxyURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetIdleAfter returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable.
func (c *EnvConfig) GetIdleAfter() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetIdleLongbuild returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable.
func (c *EnvConfig) GetIdleLongBuild() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetMaxRetriesQuietInterval returns the number of minutes no retry occurs after the maximum retry count is reached.
func (c *EnvConfig) GetMaxRetriesQuietInterval() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetMaxRetries returns the maximum number of retries to idle resp. un-idle the Jenkins service.
func (c *EnvConfig) GetMaxRetries() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetCheckInterval returns the number of minutes after which a regular idle check occurs.
func (c *EnvConfig) GetCheckInterval() int {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	i, _ := strconv.Atoi(value)
	return i
}

// GetTenantURL returns the F8 Tenant API URL as set via default, config file, or environment variable.
func (c *EnvConfig) GetTenantURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetToggleURL returns the Toggle Service URL as set via default, config file, or environment variable.
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

// GetFixedUuids returns a slice of fixed user uuids. The uuids are specified comma separated in the environment variable.
// JC_FIXED_UUIDS.
func (c *EnvConfig) GetFixedUuids() []string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	if len(value) == 0 {
		return []string{}
	}

	return strings.Split(value, ",")
}

// GetAuthURL returns the Auth API URL as set via default, config file, or environment variable
func (c *EnvConfig) GetAuthURL() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetServiceAccountId returns the service account id for the Auth service. Used to identify the Idler to the Auth service
func (c *EnvConfig) GetServiceAccountID() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetServiceAccountSecret returns the service account secret. Used to authenticate the Idler to the Auth service.
func (c *EnvConfig) GetServiceAccountSecret() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetAuthTokenKey returns the key to decrypt OpenShift API tokens obtained via the Cluster API.
func (c *EnvConfig) GetAuthTokenKey() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// GetAuthGrantType returns the fabric8-auth Grant type used while retrieving
// user account token
func (c *EnvConfig) GetAuthGrantType() string {
	callPtr, _, _, _ := runtime.Caller(0)
	value := c.getConfigValueFromEnv(util.NameOfFunction(callPtr))

	return value
}

// Verify checks whether all needed config options are set.
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

		if strings.Contains(setting.key, "SECRET") && len(value) > 0 {
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
