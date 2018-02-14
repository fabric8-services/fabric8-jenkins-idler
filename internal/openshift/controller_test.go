package openshift

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/common"
	"testing"

	"context"
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"sync"
)

var (
	tenantService    *httptest.Server
	openShiftService *httptest.Server
	controller       Controller
	origWriter       io.Writer
	testUserId       = "2e15e957-0366-4802-bf1e-0d6fe3f11bb6"
)

type mockFeatureToggle struct {
}

func (m *mockFeatureToggle) IsIdlerEnabled(uid string) (bool, error) {
	if uid == testUserId {
		return true, nil
	} else {
		return false, nil
	}
}

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

	openShiftClient := client.NewOpenShift(openShiftService.URL, "")
	tenantClient := tenant.NewTenant(tenantService.URL, "")

	features := &mockFeatureToggle{}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller = NewOpenShiftController(openShiftClient, &tenantClient, features, &mock.MockConfig{}, &wg, ctx, cancel)
}

func tearDown() {
	tenantService.Close()
	openShiftService.Close()
	log.SetOutput(origWriter)
}
