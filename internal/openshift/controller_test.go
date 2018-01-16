package openshift

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/common"
	"testing"

	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
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
var controller Controller
var origWriter io.Writer

func Test_handle_build(t *testing.T) {
	setUp(t)
	defer tearDown()

	obj := model.Object{
		Object: model.Build{
			Metadata: model.Metadata{
				Namespace: "test-namespace",
			},
		},
		Type: "MODIFIED",
	}

	ok, err := controller.HandleBuild(obj)
	assert.NoError(t, err)
	assert.True(t, ok, fmt.Sprintf("Namespace '%s' should be watched", obj.Object.Metadata.Namespace))
}

func Test_handle_deployment_config(t *testing.T) {
	setUp(t)
	defer tearDown()

	obj := model.DCObject{
		Object: model.DeploymentConfig{
			Metadata: model.Metadata{
				Namespace: "test-namespace-jenkins",
			},
			Status: model.DCStatus{
				Conditions: []model.Condition{
					{
						Type:   "Available",
						Status: "false",
					},
				},
			},
		},
		Type: "MODIFIED",
	}

	ok, err := controller.HandleDeploymentConfig(obj)
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

	openShiftClient := NewOpenShift(openShiftService.URL, "")
	tenantClient := proxyClient.NewTenant(tenantService.URL, "")

	features, err := toggles.NewUnleashToggle("http://unleash.herokuapp.com/api/")
	assert.NoError(t, err)

	controller = NewOpenShiftController(openShiftClient, &tenantClient, features, &mock.MockConfig{})
}

func tearDown() {
	tenantService.Close()
	openShiftService.Close()
	log.SetOutput(origWriter)
}
