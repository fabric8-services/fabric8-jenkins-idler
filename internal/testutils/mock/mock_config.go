package mock

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

// Config a mock implementation of the configuration.Configuration interface.
// It can be used in tests where any field can be explicitly set to return the needed value.
type Config struct {
	ProxyURL              string
	TenantURL             string
	ToggleURL             string
	IdleAfter             int
	IdleLongBuild         int
	MaxRetries            int
	MaxRetriesQuietPeriod int
	CheckInterval         int
	Debug                 bool
	FixedUuids            []string
	AuthURL               string
	ServiceAccountID      string
	ServiceAccountSecret  string
	AuthTokenKey          string
}

// GetProxyURL returns the Jenkins Proxy API URL.
func (c *Config) GetProxyURL() string {
	return c.ProxyURL
}

// GetTenantURL returns the F8 Tenant API URL.
func (c *Config) GetTenantURL() string {
	return c.TenantURL
}

// GetAuthURL returns the Auth API URL as set via default, config file, or environment variable
func (c *Config) GetAuthURL() string {
	return c.AuthURL
}

// GetToggleURL returns the Toggle Service URL.
func (c *Config) GetToggleURL() string {
	return c.ToggleURL
}

// GetIdleAfter returns the number of minutes before Jenkins is idled.
func (c *Config) GetIdleAfter() int {
	return c.IdleAfter
}

// GetIdleLongBuild returns the number of minutes before Jenkins is idled.
func (c *Config) GetIdleLongBuild() int {
	return c.IdleLongBuild
}

// GetMaxRetries returns the maximum number of retries to idle resp. un-idle the Jenkins service.
func (c *Config) GetMaxRetries() int {
	return c.MaxRetries
}

// GetMaxRetriesQuietInterval returns the number of minutes no retry occurs after the maximum retry count is reached.
func (c *Config) GetMaxRetriesQuietInterval() int {
	return c.MaxRetriesQuietPeriod
}

// GetCheckInterval returns the number of minutes after which a regular idle check occurs.
func (c *Config) GetCheckInterval() int {
	return c.CheckInterval
}

// GetDebugMode returns if debug mode should be enabled.
func (c *Config) GetDebugMode() bool {
	return c.Debug
}

// GetFixedUuids returns a slice of fixed user uuids. If set, a custom Features implementation is instantiated
// which only enabled the idler feature for the specified list of users. This is mainly used for local dev only.
func (c *Config) GetFixedUuids() []string {
	return c.FixedUuids
}

// GetServiceAccountID returns the service account id for the Auth service. Used to identify the Idler to the Auth service
func (c *Config) GetServiceAccountID() string {
	return c.ServiceAccountID
}

// GetServiceAccountSecret returns the service account secret. Used to authenticate the Idler to the Auth service.
func (c *Config) GetServiceAccountSecret() string {
	return c.ServiceAccountSecret
}

// GetAuthTokenKey returns the key to decrypt OpenShift API tokens obtained via the Cluster API.
func (c *Config) GetAuthTokenKey() string {
	return c.AuthTokenKey
}

// GetAuthGrantType returns the fabric8-auth Grant type used while retrieving
// user account token
func (c *Config) GetAuthGrantType() string {
	return "client_credentials"
}

// Verify validates the configuration and returns an error in case the configuration is missing required settings
// or contains invalid settings. If the configuration is correct nil is returned.
func (c *Config) Verify() util.MultiError {
	return util.MultiError{}
}

// String returns the current configuration as string.
func (c *Config) String() string {
	return "mockConfig"
}
