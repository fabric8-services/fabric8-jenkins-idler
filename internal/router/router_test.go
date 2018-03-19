package router

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"encoding/json"
	"net/http/httptest"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

const (
	testPortBase = 48080
)

type mockIdlerAPI struct {
}

func (i *mockIdlerAPI) Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("Info"))
	w.WriteHeader(http.StatusOK)
}

func (i *mockIdlerAPI) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("Idle"))
	w.WriteHeader(http.StatusOK)
}

func (i *mockIdlerAPI) UnIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("UnIdle"))
	w.WriteHeader(http.StatusOK)
}

func (i *mockIdlerAPI) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("IsIdle"))
	w.WriteHeader(http.StatusOK)
}

func (i *mockIdlerAPI) ClusterDNSView(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	w.Write([]byte("GetClusterDNSView"))
	w.WriteHeader(http.StatusOK)
}

func Test_all_routes_are_setup(t *testing.T) {
	router := CreateAPIRouter(&mockIdlerAPI{})

	var routes = []struct {
		route  string
		target string
	}{
		{"/api/idler/info/my-namepace", "Info"},
		{"/api/idler/info/my-namepace/", "Info"},
		{"/api/idler/idle/my-namepace", "Idle"},
		{"/api/idler/idle/my-namepace/", "Idle"},
		{"/api/idler/unidle/my-namepace", "UnIdle"},
		{"/api/idler/unidle/my-namepace/", "UnIdle"},
		{"/api/idler/isidle/my-namepace", "IsIdle"},
		{"/api/idler/isidle/my-namepace/", "IsIdle"},
		{"/api/idler/cluster", "GetClusterDNSView"},
		{"/api/idler/cluster/", "GetClusterDNSView"},

		{"/api/idler/foo", "404 page not found\n"},
		{"/api/idler/builds/foo/bar", "404 page not found\n"},
	}

	for _, testRoute := range routes {
		w := new(mock.ResponseWriter)

		req, _ := http.NewRequest("GET", testRoute.route, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, testRoute.target, w.GetBody(), fmt.Sprintf("Routing failed for %s", testRoute.route))
	}
}

func Test_router_start(t *testing.T) {
	testPort := testPortBase + 1
	log.SetOutput(ioutil.Discard)

	testLogger, hook := test.NewNullLogger()
	routerLogger = testLogger.WithFields(log.Fields{"component": "router"})

	assert.True(t, isTCPPortAvailable(testPort), fmt.Sprintf("Port '%d' should be free.", testPort))

	router := NewRouterWithPort(CreateAPIRouter(&mockIdlerAPI{}), testPort)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router.Start(ctx, &wg, cancel)

	// we need to give a bit time for the server to come up
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/idler/info/foo", testPort))
	assert.NoError(t, err, "The call to the API should have succeeded.")
	body, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "Info", string(body), "Unexpected result from HTTP request")

	go func() {
		// Cancel the operation after 2 second.
		time.Sleep(2 * time.Second)
		cancel()
	}()

	wg.Wait()

	assert.Equal(t, fmt.Sprintf("Shutting down API router on port %d.", testPort), hook.LastEntry().Message)
}

func Test_cluster_dns_view(t *testing.T) {
	testPort := testPortBase + 2
	log.SetOutput(ioutil.Discard)
	assert.True(t, isTCPPortAvailable(testPort), fmt.Sprintf("Port '%d' should be free.", testPort))

	dummyCluster := cluster.Cluster{
		APIURL: "http://localhost",
		AppDNS: "example.com",
	}
	clusterView := cluster.NewView([]cluster.Cluster{dummyCluster})
	idlerAPI := api.NewIdlerAPI(openshift.NewUserIdlerMap(), clusterView)
	router := NewRouterWithPort(CreateAPIRouter(idlerAPI), testPort)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router.Start(ctx, &wg, cancel)

	// we need to give a bit time for the server to come up
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/idler/cluster", testPort))
	assert.NoError(t, err, "The call to the API should have succeeded.")
	body, err := ioutil.ReadAll(resp.Body)
	expectedResponse := `[{"APIURL":"http://localhost","AppDNS":"example.com"}]
`
	assert.Equal(t, expectedResponse, string(body), "Unexpected result from HTTP request")
	cancel()

	wg.Wait()
}

func Test_openshift_url_parameter_is_used(t *testing.T) {
	testPort := testPortBase + 3
	log.SetOutput(ioutil.Discard)
	assert.True(t, isTCPPortAvailable(testPort), fmt.Sprintf("Port '%d' should be free.", testPort))

	// a dummy HTTP server for the OpenShift API request which is going to occur
	openShiftAPITestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(model.DeploymentConfig{})
	}))
	defer openShiftAPITestServer.Close()

	// setup the router
	dummyCluster := cluster.Cluster{
		APIURL: util.EnsureSuffix(openShiftAPITestServer.URL, "/"),
		Token:  "mysecret",
	}
	clusterView := cluster.NewView([]cluster.Cluster{dummyCluster})
	idlerAPI := api.NewIdlerAPI(openshift.NewUserIdlerMap(), clusterView)
	router := NewRouterWithPort(CreateAPIRouter(idlerAPI), testPort)

	// start the router
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router.Start(ctx, &wg, cancel)

	// we need to give a bit time for the server to come up
	time.Sleep(1 * time.Second)

	// make the test request
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/api/idler/isidle/foo", testPort), nil)
	assert.NoError(t, err)

	q := req.URL.Query()
	q.Add(api.OpenShiftAPIParam, util.EnsureSuffix(openShiftAPITestServer.URL, "/"))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)

	assert.NoError(t, err, "The call to the API should have succeeded.")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected HTTP status code")
	cancel()

	wg.Wait()
}

func isTCPPortAvailable(port int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
