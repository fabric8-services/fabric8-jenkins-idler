package openshift

import (
	"math/rand"
	"strconv"
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
			Status: model.Status{
				Phase: "Running",
			},
		},
		Type: "MODIFIED",
	}

	// List of build phases here
	// https://github.com/openshift/origin/blob/1017d1d8ca3611267e3993742a2c4fb06f65e449/pkg/build/apis/build/types.go#L403
	// List of event types over here
	// https://github.com/kubernetes/kubernetes/blob/f9acfd8e384488d2216b18196152dcb7b3cc92d8/pkg/watch/json/types.go#L33

	// Active builds
	testWithPhaseChange(t, obj, "New", "ADDED")
	testWithPhaseChange(t, obj, "Pending", "MODIFIED")
	testWithPhaseChange(t, obj, "Running", "MODIFIED")

	// Done builds
	testWithPhaseChange(t, obj, "Complete", "MODIFIED")
	testWithPhaseChange(t, obj, "Failed", "MODIFIED")
	testWithPhaseChange(t, obj, "Cancelled", "MODIFIED")
	testWithPhaseChange(t, obj, "Error", "ERROR")
	testWithPhaseChange(t, obj, "Error", "DELETED")

	// Above all are independent events which don't occur in reality
	// In real situations builds will transition from one state to the other
	testWithStateChange(t, obj)
}

func testWithStateChange(t *testing.T, obj model.Object) {
	obj.Type = "MODIFIED"
	obj.Object.Status.Phase = "Running"
	obj.Object.Metadata.Name = "#1"

	ci := controller.(*controllerImpl)
	err := controller.HandleBuild(obj)
	assert.NoError(t, err)

	userIdler := ci.userIdlerForNamespace("test-namespace").GetChannel()
	userBefore := <-userIdler
	assert.True(t, userBefore.HasActiveBuilds())
	assert.Equal(t, obj.Object.Status.Phase, userBefore.ActiveBuild.Status.Phase)

	obj.Object.Status.Phase = "Complete"

	err = controller.HandleBuild(obj)
	assert.NoError(t, err)

	userAfter := <-userIdler
	assert.False(t, userAfter.HasActiveBuilds())
	assert.Equal(t, obj.Object.Status.Phase, userAfter.DoneBuild.Status.Phase)

	// The build that was previously active has now been moved to done
	assert.Equal(t, userBefore.ActiveBuild.Metadata.Name, userAfter.DoneBuild.Metadata.Name)
}

func testWithPhaseChange(t *testing.T, obj model.Object, phase string, eventType string) {
	obj.Type = eventType
	obj.Object.Status.Phase = phase
	obj.Object.Metadata.Name = strconv.Itoa(rand.Intn(1000))

	ci := controller.(*controllerImpl)
	err := controller.HandleBuild(obj)
	assert.NoError(t, err)

	userIdler := ci.userIdlerForNamespace("test-namespace")
	userAfter := <-userIdler.GetChannel()

	if isActive(obj) {
		assert.True(t, userAfter.HasActiveBuilds())
		assert.Equal(t, obj.Object.Status.Phase, userAfter.ActiveBuild.Status.Phase)
	} else {
		assert.False(t, userAfter.HasActiveBuilds())
		assert.Equal(t, obj.Object.Status.Phase, userAfter.DoneBuild.Status.Phase)
	}

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
	controller = NewController(ctx, "", "", userIdlers, tenantService, features, &mock.Config{}, &wg, cancel)
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
