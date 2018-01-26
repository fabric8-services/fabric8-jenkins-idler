package mock

import "github.com/fabric8-services/fabric8-jenkins-idler/internal/util"

type MockConfig struct {
}

func (c *MockConfig) GetOpenShiftToken() string {
	return ""
}

func (c *MockConfig) GetOpenShiftURL() string {
	return ""
}

func (c *MockConfig) GetProxyURL() string {
	return ""
}

func (c *MockConfig) GetTenantURL() string {
	return ""
}

func (c *MockConfig) GetAuthToken() string {
	return ""
}

func (c *MockConfig) GetToggleURL() string {
	return ""
}

func (c *MockConfig) GetIdleAfter() int {
	return 10
}

func (c *MockConfig) GetUnIdleRetry() int {
	return 10
}

func (c *MockConfig) GetDebugMode() bool {
	return false
}

func (c *MockConfig) Verify() util.MultiError {
	return util.MultiError{}
}

func (c *MockConfig) String() string {
	return "mockConfig"
}
