package openshift

import (
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/common"

	"context"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"sync"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	tenantService    *httptest.Server
	openShiftService *httptest.Server
	controller       Controller
	origWriter       io.Writer
	testUserID       = "2e15e957-0366-4802-bf1e-0d6fe3f11bb6"
)

type mockFeatureToggle struct {
}

func (m *mockFeatureToggle) IsIdlerEnabled(uid string) (bool, error) {
	if uid == testUserID {
		return true, nil
	}

	return false, nil
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

	err := controller.HandleBuild(obj)
	assert.NoError(t, err)
}

func TestHandleBuildChannelLength(t *testing.T) {
	setUp(t)
	defer tearDown()

	tests := []struct {
		name          string
		object        model.Object
		channelLength int
	}{
		{
			name: "both, build is with different last active/done phase, and active and done build name is the same, length should still be 1",
			object: model.Object{
				Object: model.Build{
					Status: model.Status{
						Phase: "NotNew",
					},
				},
			},
			channelLength: 1,
		},
		{
			name: "both, build is with different last active/done name, and active and done build name is the same, length should still be 1",
			object: model.Object{
				Object: model.Build{
					Metadata: model.Metadata{
						Name: "NotEmpty",
					},
				},
			},
			channelLength: 1,
		},
	}

	for _, test := range tests {
		t.Logf("Running test: %v", test.name)

		err := controller.HandleBuild(test.object)
		assert.NoError(t, err)

		ns := test.object.Object.Metadata.Namespace
		ci := controller.(*controllerImpl)
		userIdler := ci.userIdlerForNamespace(ns)

		if userIdler == nil {
			t.Errorf("expected user-idler to be created")
			return
		}

		userChannel := userIdler.GetChannel()
		if test.channelLength != len(userChannel) {
			t.Errorf("Expected channel length to be %v, but got %v", test.channelLength, len(userChannel))
		}
		emptyChannel(userChannel)
	}
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

	err := controller.HandleDeploymentConfig(obj)
	assert.NoError(t, err)
}

func TestHandleDeploymentConfigChannelLength(t *testing.T) {
	setUp(t)
	defer tearDown()

	tests := []struct {
		name          string
		object        model.DCObject
		channelLength int
	}{
		{
			name: "both new DC and available condition is true, length should still be 1",
			object: model.DCObject{
				Object: model.DeploymentConfig{
					Metadata: model.Metadata{
						Namespace:  "test-namespace-jenkins",
						Generation: 1,
					},
					Spec: model.Spec{
						Replicas: 1,
					},
					Status: model.DCStatus{
						ObservedGeneration: 2,
						Conditions: []model.Condition{
							{
								Type:   availableCond,
								Status: "true",
							},
						},
					},
				},
			},
			channelLength: 1,
		},
	}

	for _, test := range tests {
		t.Logf("Running test: %v", test.name)

		err := controller.HandleDeploymentConfig(test.object)
		assert.NoError(t, err)

		ns := test.object.Object.Metadata.Namespace[:len(test.object.Object.Metadata.Namespace)-len(jenkinsNamespaceSuffix)]
		ci := controller.(*controllerImpl)
		userIdler := ci.userIdlerForNamespace(ns)

		if userIdler == nil {
			t.Errorf("Expected user-idler to be created")
			return
		}

		userChannel := userIdler.GetChannel()
		if test.channelLength != len(userChannel) {
			t.Errorf("Expected channel length to be %v, but got %v", test.channelLength, len(userChannel))
		}
		emptyChannel(userChannel)
	}
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

	tenantService := tenant.NewTenantService(tenantService.URL, "")

	features := &mockFeatureToggle{}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	userIdlers := NewUserIdlerMap()
	disabledUsers := model.NewStringSet()
	controller = NewController(ctx, "", "", userIdlers, tenantService, features, &mock.Config{}, &wg, cancel, disabledUsers)
}

func emptyChannel(ch chan model.User) {
	for len(ch) > 0 {
		<-ch
	}
}

func tearDown() {
	tenantService.Close()
	openShiftService.Close()
	log.SetOutput(origWriter)
}
