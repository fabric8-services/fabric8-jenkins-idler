package openshiftcontroller_test

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/testutils"
	"testing"
	"time"

	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshiftcontroller"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	pc "github.com/fabric8-services/fabric8-jenkins-proxy/clients"
)

func TestToggleHandleBuild(t *testing.T) {
	ts := testutils.MockServer(testutils.TenantData())
	defer ts.Close()
	os := testutils.MockServer(testutils.DCJenkinsDataIdle())
	defer os.Close()
	o := ic.NewOpenShift(os.URL, "")
	tc := pc.NewTenant(ts.URL, "")

	toggles.Init("jenkins-idler", "http://unleash.herokuapp.com/api/")

	oc := openshiftcontroller.NewOpenShiftController(o, tc, 0, 10, []string{}, "", 0, true)

	obj := ic.Object{
		Object: ic.Build{
			Metadata: ic.Metadata{
				Namespace: "test-namespace",
			},
		},
		Type: "MODIFIED",
	}

	for i := 0; i < 10; i++ {
		if toggles.IsReady() {
			break
		}
		time.Sleep(1 * time.Second)
	}

	ok, err := oc.HandleBuild(obj)
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Errorf("Do not watch %s", obj.Object.Metadata.Namespace)
	}
}

func TestToggleHandleDC(t *testing.T) {
	ts := testutils.MockServer(testutils.TenantData())
	defer ts.Close()
	os := testutils.MockServer(testutils.DCJenkinsDataIdle())
	defer os.Close()
	o := ic.NewOpenShift(os.URL, "")
	tc := pc.NewTenant(ts.URL, "")

	toggles.Init("jenkins-idler", "http://unleash.herokuapp.com/api/")

	oc := openshiftcontroller.NewOpenShiftController(o, tc, 0, 10, []string{}, "", 0, true)

	obj := ic.DCObject{
		Object: ic.DeploymentConfig{
			Metadata: ic.Metadata{
				Namespace: "test-namespace-jenkins",
			},
			Status: ic.DCStatus{
				Conditions: []ic.Condition{
					{
						Type:   "Available",
						Status: "false",
					},
				},
			},
		},
		Type: "MODIFIED",
	}

	for i := 0; i < 10; i++ {
		if toggles.IsReady() {
			break
		}
		time.Sleep(1 * time.Second)
	}

	ok, err := oc.HandleDeploymentConfig(obj)
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Errorf("Do not watch %s", obj.Object.Metadata.Namespace)
	}
}
