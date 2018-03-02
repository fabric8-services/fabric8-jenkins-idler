package mock

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
)

// OpenShiftClient is a client for OpenShift API
// It is a mock implementation of client.OpenShiftClient.
type OpenShiftClient struct {
	IdleState       int
	IdleCallCount   int
	UnIdleCallCount int
}

// Idle mocks Idle method of client.OpenShiftClient.
// It increases IdleCallCount by 1.
func (c *OpenShiftClient) Idle(namespace string, service string) error {
	c.IdleCallCount++
	return nil
}

// UnIdle mocks UnIdle method of client.OpenShiftClient.
// It increases UnIdleCallCount by 1.
func (c *OpenShiftClient) UnIdle(namespace string, service string) error {
	c.UnIdleCallCount++
	return nil
}

// IsIdle mocks IsIdle method of client.OpenShiftClient.
func (c *OpenShiftClient) IsIdle(namespace string, service string) (int, error) {
	return c.IdleState, nil
}

// GetRoute mocks GetRoute method of client.OpenShiftClient.
// It always return ("", true, nil).
func (c *OpenShiftClient) GetRoute(n string, s string) (r string, tls bool, err error) {
	return "", true, nil
}

// GetAPIURL mocks GetAPIURL method of client.OpenShiftClient.
// It always returns "".
func (c *OpenShiftClient) GetAPIURL() string {
	return ""
}

// WatchBuilds mocks WatchBuilds method of client.OpenShiftClient.
// It always returns nil.
func (c *OpenShiftClient) WatchBuilds(namespace string, buildType string, callback func(model.Object) error) error {
	return nil
}

// WatchDeploymentConfigs mocks WatchDeploymentConfigs method of client.OpenShiftClient.
// It always returns nil.
func (c *OpenShiftClient) WatchDeploymentConfigs(namespace string, nsSuffix string, callback func(model.DCObject) error) error {
	return nil
}

// ResetCounts resets calls made to the idler(idle/unidle) to 0.
func (c *OpenShiftClient) ResetCounts() {
	c.UnIdleCallCount = 0
	c.IdleCallCount = 0
}

// String return name of the OpenShiftClient.
func (c *OpenShiftClient) String() string {
	return "MockOpenShiftClient"
}
