package router

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

type mockResponseWriter struct {
	buffer bytes.Buffer
}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	m.buffer.Write(p)
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	m.buffer.WriteString(s)
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func (m *mockResponseWriter) GetBody() string {
	return m.buffer.String()
}

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

func (i *mockIdlerAPI) GetRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	w.Write([]byte("GetRoute"))
	w.WriteHeader(http.StatusOK)
}

func Test_all_routes_are_setup(t *testing.T) {
	router := CreateAPIRouter(&mockIdlerAPI{})

	var routes = []struct {
		route  string
		target string
	}{
		{"/iapi/idler/info/my-namepace", "Info"},
		{"/iapi/idler/info/my-namepace/", "Info"},
		{"/iapi/idler/idle/my-namepace", "Idle"},
		{"/iapi/idler/idle/my-namepace/", "Idle"},
		{"/iapi/idler/unidle/my-namepace", "UnIdle"},
		{"/iapi/idler/unidle/my-namepace/", "UnIdle"},
		{"/iapi/idler/isidle/my-namepace", "IsIdle"},
		{"/iapi/idler/isidle/my-namepace/", "IsIdle"},

		{"/iapi/idler/foo", "404 page not found\n"},
		{"/iapi/idler/builds/foo/bar", "404 page not found\n"},
	}

	for _, testRoute := range routes {
		w := new(mockResponseWriter)

		req, _ := http.NewRequest("GET", testRoute.route, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, testRoute.target, w.GetBody(), fmt.Sprintf("Routing failed for %s", testRoute.route))
	}
}

func Test_router_start(t *testing.T) {
	//log.SetOutput(ioutil.Discard)

	testLogger, hook := test.NewNullLogger()
	routerLogger = testLogger.WithFields(log.Fields{"component": "router"})

	testPort := 48080

	assert.True(t, isTCPPortAvailable(testPort), fmt.Sprintf("Port '%d' should be free.", testPort))

	router := NewRouterWithPort(CreateAPIRouter(&mockIdlerAPI{}), testPort)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router.Start(ctx, &wg, cancel)

	// we need to give a bit time for the server to come up
	time.Sleep(1 * time.Second)
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/iapi/idler/info/foo", testPort))
	assert.NoError(t, err, "The call to the API should have succeeded.")
	body, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "Info", string(body), "Unexpected result from HTTP request")

	go func() {
		// Cancel the operation after 2 second.
		time.Sleep(2 * time.Second)
		cancel()
	}()

	wg.Wait()

	assert.Equal(t, "Shutting down API router on port 48080.", hook.LastEntry().Message)
}

func isTCPPortAvailable(port int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
