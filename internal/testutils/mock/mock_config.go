package mock

import "github.com/fabric8-services/fabric8-jenkins-idler/internal/util"

// Config contains all configuration detail (prerequisite as well) to run idler.
// It implements configuration.Configuration
type Config struct {
	OpenShiftToken string
	OpenShiftURL   string
	ProxyURL       string
	TenantURL      string
	AuthToken      string
	ToggleURL      string
	IdleAfter      int
	MaxRetries     int
	CheckInterval  int
	Debug          bool
	FixedUuids     []string
}

// GetOpenShiftToken returns the OpenShift token.
func (c *Config) GetOpenShiftToken() string {
	return c.OpenShiftToken
}

// GetOpenShiftURL returns the OpenShift API URL.
func (c *Config) GetOpenShiftURL() string {
	return c.OpenShiftURL
}

// GetProxyURL returns the Jenkins Proxy API URL.
func (c *Config) GetProxyURL() string {
	return c.ProxyURL
}

// GetTenantURL returns the F8 Tenant API URL.
func (c *Config) GetTenantURL() string {
	return c.TenantURL
}

// GetAuthToken returns the Auth token.
func (c *Config) GetAuthToken() string {
	return c.AuthToken
}

// GetToggleURL returns the Toggle Service URL.
func (c *Config) GetToggleURL() string {
	return c.ToggleURL
}

// GetIdleAfter returns the number of minutes before Jenkins is idled.
func (c *Config) GetIdleAfter() int {
	return c.IdleAfter
}

// GetMaxRetries returns the maximum number of retries to idle resp. un-idle the Jenkins service.
func (c *Config) GetMaxRetries() int {
	return c.MaxRetries
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

// Verify validates the configuration and returns an error in case the configuration is missing required settings
// or contains invalid settings. If the configuration is correct nil is returned.
func (c *Config) Verify() util.MultiError {
	return util.MultiError{}
}

// String returns the current configuration as string.
func (c *Config) String() string {
	return "mockConfig"
}
