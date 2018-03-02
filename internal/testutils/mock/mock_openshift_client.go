package mock

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
)

// OpenShiftClient is a client for OpenShift API
// It implements client.OpenShiftClient
type OpenShiftClient struct {
	IdleState       int
	IdleCallCount   int
	UnIdleCallCount int
}

// Idle forces a service in OpenShift namespace to idle
func (c *OpenShiftClient) Idle(namespace string, service string) error {
	c.IdleCallCount++
	return nil
}

// UnIdle forces a service in OpenShift namespace to start
func (c *OpenShiftClient) UnIdle(namespace string, service string) error {
	c.UnIdleCallCount++
	return nil
}

//IsIdle returns `JenkinsIdled` if a service in OpenShit namespace is idled,
//`JenkinsStarting` if it is in the process of scaling up, `JenkinsRunning`
//if it is fully up
func (c *OpenShiftClient) IsIdle(namespace string, service string) (int, error) {
	return c.IdleState, nil
}

//GetRoute collects object for a given namespace and route name and returns
//url to reach it and if the route has enabled TLS
func (c *OpenShiftClient) GetRoute(n string, s string) (r string, tls bool, err error) {
	return "", true, nil
}

// GetAPIURL returns API Url for OpenShift cluster
func (c *OpenShiftClient) GetAPIURL() string {
	return ""
}

// WatchBuilds consumes stream of build events from OpenShift and calls callback to process them
func (c *OpenShiftClient) WatchBuilds(namespace string, buildType string, callback func(model.Object) error) error {
	return nil
}

// WatchDeploymentConfigs consumes stream of DC events from OpenShift and calls callback to process them
func (c *OpenShiftClient) WatchDeploymentConfigs(namespace string, nsSuffix string, callback func(model.DCObject) error) error {
	return nil
}

// ResetCounts resets calls made to the idler(idle/unidle) to 0
func (c *OpenShiftClient) ResetCounts() {
	c.UnIdleCallCount = 0
	c.IdleCallCount = 0
}

// String return name of the OpenShiftClient
func (c *OpenShiftClient) String() string {
	return "MockOpenShiftClient"
}
