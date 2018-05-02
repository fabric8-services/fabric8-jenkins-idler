package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
)

type JSError struct {
	Error string
}

type ReqFuncType func(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

func Test_success(t *testing.T) {
	mosc := &mock.OpenShiftClient{}
	mockidle := idler{
		openShiftClient: mosc,
		clusterView:     &mock.ClusterView{},
		tenantService:   &mock.TenantService{},
	}
	functions := []ReqFuncType{mockidle.Idle, mockidle.UnIdle, mockidle.IsIdle}

	params := httprouter.Params{
		httprouter.Param{Key: "namespace", Value: "foobar"},
	}

	for _, function := range functions {
		reader, _ := http.NewRequest("GET", "/", nil)
		q := reader.URL.Query()
		q.Add(OpenShiftAPIParam, "http://localhost")
		reader.URL.RawQuery = q.Encode()

		writer := &mock.ResponseWriter{}
		function(writer, reader, params)

		require.Equal(t, http.StatusOK, writer.WriterStatus, fmt.Sprintf("Bad Error Code: %d", writer.WriterStatus))
		require.Equal(t, mosc.IdleCallCount, 2, fmt.Sprintf("Idle was not called for 2 times but %d", mosc.IdleCallCount))
	}
}

func Test_fail(t *testing.T) {
	mockidle := idler{
		openShiftClient: &mock.OpenShiftClient{},
		clusterView:     &mock.ClusterView{},
	}
	functions := []ReqFuncType{mockidle.Idle, mockidle.UnIdle, mockidle.IsIdle}
	for _, function := range functions {
		reader, _ := http.NewRequest("GET", "/", nil)
		writer := &mock.ResponseWriter{}
		function(writer, reader, nil)

		require.Equal(t, http.StatusBadRequest, writer.WriterStatus, fmt.Sprintf("Bad Error Code: %d", writer.WriterStatus))

		jserror := &JSError{}
		_ = json.Unmarshal(writer.Buffer.Bytes(), &jserror)
		require.Equal(t, "OpenShift API URL needs to be specified", jserror.Error, fmt.Sprintf("Unexpected error output: %s", jserror.Error))
	}

	idleError := "Error when Idling"
	mockidle = idler{
		openShiftClient: &mock.OpenShiftClient{
			IdleError: idleError,
		},
		clusterView:   &mock.ClusterView{},
		tenantService: &mock.TenantService{},
	}
	functions = []ReqFuncType{mockidle.Idle, mockidle.UnIdle, mockidle.IsIdle}
	params := httprouter.Params{
		httprouter.Param{Key: "namespace", Value: "foobar"},
	}

	for _, function := range functions {
		req, _ := http.NewRequest("GET", "/", nil)
		query := req.URL.Query()
		query.Add(OpenShiftAPIParam, "http://localhost")
		req.URL.RawQuery = query.Encode()

		writer := &mock.ResponseWriter{}
		function(writer, req, params)

		jserror := &JSError{}
		_ = json.Unmarshal(writer.Buffer.Bytes(), &jserror)
		require.Equal(t, idleError, jserror.Error, fmt.Sprintf("Unexpected error output: %s", jserror.Error))
	}
}
