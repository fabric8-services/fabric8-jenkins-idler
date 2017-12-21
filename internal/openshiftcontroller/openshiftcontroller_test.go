package openshiftcontroller

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/common"
	"testing"

	"fmt"
	idlerClient "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	proxyClient "github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http/httptest"
)

var tenantService *httptest.Server
var openShiftService *httptest.Server
var openShiftController *OpenShiftController
var origWriter io.Writer

func Test_handle_build(t *testing.T) {
	setUp(t)
	defer tearDown()

	obj := idlerClient.Object{
		Object: idlerClient.Build{
			Metadata: idlerClient.Metadata{
				Namespace: "test-namespace",
			},
		},
		Type: "MODIFIED",
	}

	ok, err := openShiftController.HandleBuild(obj)
	assert.NoError(t, err)
	assert.True(t, ok, fmt.Sprintf("Namespace '%s' should be watched", obj.Object.Metadata.Namespace))
}

func Test_handle_deployment_config(t *testing.T) {
	setUp(t)
	defer tearDown()

	obj := idlerClient.DCObject{
		Object: idlerClient.DeploymentConfig{
			Metadata: idlerClient.Metadata{
				Namespace: "test-namespace-jenkins",
			},
			Status: idlerClient.DCStatus{
				Conditions: []idlerClient.Condition{
					{
						Type:   "Available",
						Status: "false",
					},
				},
			},
		},
		Type: "MODIFIED",
	}

	ok, err := openShiftController.HandleDeploymentConfig(obj)
	assert.NoError(t, err)
	assert.True(t, ok, fmt.Sprintf("Namespace '%s' should be watched", obj.Object.Metadata.Namespace))
}

func setUp(t *testing.T) {
	origWriter = log.StandardLogger().Out
	log.SetOutput(ioutil.Discard)

	tenantData, err := ioutil.ReadFile("../testutils/testdata/tenant.json")
	if err != nil {
		assert.NoError(t, err)
	}

	tenantService = common.MockServer(tenantData)

	deploymentConfigData, err := ioutil.ReadFile("../testutils/testdata/deploymentConfig.json")
	if err != nil {
		assert.NoError(t, err)
	}
	openShiftService = common.MockServer(deploymentConfigData)

	o := idlerClient.NewOpenShift(openShiftService.URL, "")
	tc := proxyClient.NewTenant(tenantService.URL, "")

	features, err := toggles.NewUnleashToggle("http://unleash.herokuapp.com/api/")
	assert.NoError(t, err)

	openShiftController = NewOpenShiftController(o, tc, 0, 10, []string{}, "", 0, true, features)
}

func tearDown() {
	tenantService.Close()
	openShiftService.Close()
	log.SetOutput(origWriter)
}
