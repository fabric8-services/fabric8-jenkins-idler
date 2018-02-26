package mock

import "github.com/fabric8-services/fabric8-jenkins-idler/internal/util"

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

func (c *Config) GetOpenShiftToken() string {
	return c.OpenShiftToken
}

func (c *Config) GetOpenShiftURL() string {
	return c.OpenShiftURL
}

func (c *Config) GetProxyURL() string {
	return c.ProxyURL
}

func (c *Config) GetTenantURL() string {
	return c.TenantURL
}

func (c *Config) GetAuthToken() string {
	return c.AuthToken
}

func (c *Config) GetToggleURL() string {
	return c.ToggleURL
}

func (c *Config) GetIdleAfter() int {
	return c.IdleAfter
}

func (c *Config) GetMaxRetries() int {
	return c.MaxRetries
}

func (c *Config) GetCheckInterval() int {
	return c.CheckInterval
}

func (c *Config) GetDebugMode() bool {
	return c.Debug
}

func (c *Config) GetFixedUuids() []string {
	return c.FixedUuids
}

func (c *Config) Verify() util.MultiError {
	return util.MultiError{}
}

func (c *Config) String() string {
	return "mockConfig"
}
