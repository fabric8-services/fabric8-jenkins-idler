package mock

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
)

type OpenShiftClient struct {
	IdleState       int
	IdleCallCount   int
	UnIdleCallCount int
}

func (c *OpenShiftClient) Idle(namespace string, service string) error {
	c.IdleCallCount++
	return nil
}

func (c *OpenShiftClient) UnIdle(namespace string, service string) error {
	c.UnIdleCallCount++
	return nil
}
func (c *OpenShiftClient) IsIdle(namespace string, service string) (int, error) {
	return c.IdleState, nil
}

func (c *OpenShiftClient) GetRoute(n string, s string) (r string, tls bool, err error) {
	return "", true, nil
}

func (c *OpenShiftClient) GetAPIURL() string {
	return ""
}

func (c *OpenShiftClient) WatchBuilds(namespace string, buildType string, callback func(model.Object) error) error {
	return nil
}

func (c *OpenShiftClient) WatchDeploymentConfigs(namespace string, nsSuffix string, callback func(model.DCObject) error) error {
	return nil
}

func (c *OpenShiftClient) ResetCounts() {
	c.UnIdleCallCount = 0
	c.IdleCallCount = 0
}

func (c *OpenShiftClient) String() string {
	return "MockOpenShiftClient"
}
