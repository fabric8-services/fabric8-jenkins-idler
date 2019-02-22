package configuration

import (
	"fmt"
	"strings"

	errs "github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

const (
	// Constants for viper variable names. Will be used to set
	// default values as well as to get each value
	proxyURL                = "JC_JENKINS_PROXY_API_URL"
	tenantURL               = "JC_F8TENANT_API_URL"
	toggleURL               = "JC_TOGGLE_API_URL"
	authURL                 = "JC_AUTH_URL"
	serviceAccountID        = "JC_SERVICE_ACCOUNT_ID"
	serviceAccountSecret    = "JC_SERVICE_ACCOUNT_SECRET"
	authTokenKey            = "JC_AUTH_TOKEN_KEY"
	authGrantType           = "JC_AUTH_GRANT_TYPE"
	idleAfter               = "JC_IDLE_AFTER"
	idleLongBuild           = "JC_IDLE_LONG_BUILD"
	maxRetries              = "JC_MAX_RETRIES"
	maxRetriesQuietInterval = "JC_MAX_RETRIES_QUIET_INTERVAL"
	checkInterval           = "JC_CHECK_INTERVAL"
	debugMode               = "JC_DEBUG_MODE"
	fixedUuids              = "JC_FIXED_UUIDS"

	defaultIdleLongBuild           = 3
	defaultIdleAfter               = 45
	defaultMaxRetries              = 10
	defaultMaxRetriesQuietInterval = 30
	defaultCheckInterval           = 15
)

// New creates a configuration reader object using a configurable configuration
// file path.
func New(configFilePath string) (Configuration, error) {
	c := Config{
		v: viper.New(),
	}
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()

	if configFilePath != "" {
		c.v.SetConfigType("yaml")
		c.v.SetConfigFile(configFilePath)
		err := c.v.ReadInConfig() // Find and read the config file
		if err != nil {           // Handle errors reading the config file
			return nil, errs.Errorf("Fatal error config file: %s \n", err)
		}
	}
	return &c, nil
}

// Config encapsulates the Viper configuration registry which stores the
// configuration data in-memory.
type Config struct {
	v *viper.Viper
}

func (c *Config) setConfigDefaults() {

	c.v.SetDefault(proxyURL, "")
	c.v.SetDefault(tenantURL, "")
	c.v.SetDefault(toggleURL, "")
	c.v.SetDefault(authURL, "authur")
	c.v.SetDefault(serviceAccountID, "")
	c.v.SetDefault(serviceAccountSecret, "")
	c.v.SetDefault(authTokenKey, "")
	c.v.SetDefault(authGrantType, "client_credentials")
	c.v.SetDefault(idleAfter, defaultIdleAfter)
	c.v.SetDefault(idleLongBuild, defaultIdleLongBuild)
	c.v.SetDefault(maxRetries, defaultMaxRetries)
	c.v.SetDefault(maxRetriesQuietInterval, defaultMaxRetriesQuietInterval)
	c.v.SetDefault(checkInterval, defaultCheckInterval)

	c.v.SetDefault(debugMode, false)
	c.v.SetDefault(fixedUuids, []string{})
}

// GetDebugMode returns `true` if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func (c *Config) GetDebugMode() bool {
	return c.v.GetBool(debugMode)
}

// GetProxyURL returns the Jenkins Proxy API URL as set via default, config file, or environment variable.
func (c *Config) GetProxyURL() string {
	return c.v.GetString(proxyURL)
}

// GetTenantURL returns the F8 Tenant API URL as set via default, config file, or environment variable.
func (c *Config) GetTenantURL() string {
	return c.v.GetString(tenantURL)
}

// GetToggleURL returns the Toggle Service URL as set via default, config file, or environment variable.
func (c *Config) GetToggleURL() string {
	return c.v.GetString(toggleURL)
}

// GetAuthURL returns the Auth API URL as set via default, config file, or environment variable
func (c *Config) GetAuthURL() string {
	return c.v.GetString(authURL)
}

// GetServiceAccountID returns the service account id for the Auth service. Used to identify the Idler to the Auth service
func (c *Config) GetServiceAccountID() string {
	return c.v.GetString(serviceAccountID)
}

// GetServiceAccountSecret returns the service account secret. Used to authenticate the Idler to the Auth service.
func (c *Config) GetServiceAccountSecret() string {
	return c.v.GetString(serviceAccountSecret)
}

// GetAuthTokenKey returns the key to decrypt OpenShift API tokens obtained via the Cluster API.
func (c *Config) GetAuthTokenKey() string {
	return c.v.GetString(authTokenKey)
}

// GetAuthGrantType returns the fabric8-auth Grant type used while retrieving
// user account token
func (c *Config) GetAuthGrantType() string {
	return c.v.GetString(authGrantType)
}

// GetIdleAfter returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable.
func (c *Config) GetIdleAfter() int {
	return c.v.GetInt(idleAfter)
}

// GetIdleLongBuild returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable.
func (c *Config) GetIdleLongBuild() int {
	return c.v.GetInt(idleLongBuild)
}

// GetMaxRetries returns the maximum number of retries to idle resp. un-idle the Jenkins service.
func (c *Config) GetMaxRetries() int {
	return c.v.GetInt(maxRetries)
}

// GetMaxRetriesQuietInterval returns the number of minutes no retry occurs after the maximum retry count is reached.
func (c *Config) GetMaxRetriesQuietInterval() int {
	return c.v.GetInt(maxRetriesQuietInterval)
}

// GetCheckInterval returns the number of minutes after which a regular idle check occurs.
func (c *Config) GetCheckInterval() int {
	return c.v.GetInt(checkInterval)
}

// GetFixedUuids returns a slice of fixed user uuids.
// The uuids are whitespace separated in the environment variable.
// JC_FIXED_UUIDS.
func (c *Config) GetFixedUuids() []string {
	return c.v.GetStringSlice(fixedUuids)
}

// String returns string representation of configuration
func (c *Config) String() string {
	all := c.v.AllSettings()
	for k := range all {
		// don't echo tokens or secret
		if strings.Contains(k, "TOKEN") ||
			strings.Contains(k, "token") {
			all[k] = "***"
		}

		if strings.Contains(k, "SECRET") ||
			strings.Contains(k, "secret") {
			all[k] = "***"
		}
	}
	return fmt.Sprintf("%v", all)
}

// Verify checks whether all needed config options are set.
func (c *Config) Verify() util.MultiError {
	config := c.v.AllSettings()
	var errors util.MultiError
	for k, v := range config {
		switch strings.ToUpper(k) {
		case proxyURL:
			continue
		case tenantURL:
			continue
		case toggleURL:
			continue
		case authURL:
			errors.Collect(util.IsURL(v, k))
		case serviceAccountID:
			continue
		case serviceAccountSecret:
			continue
		case authTokenKey:
			continue
		case authGrantType:
			errors.Collect(util.IsNotEmpty(v, k))
		}
	}
	return errors
}
