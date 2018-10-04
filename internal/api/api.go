package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	pidler "github.com/fabric8-services/fabric8-jenkins-idler/internal/idler"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-idler/metric"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

const (
	// OpenShiftAPIParam is the parameter name under which the OpenShift cluster API URL is passed using
	// Idle, UnIdle and IsIdle.
	OpenShiftAPIParam = "openshift_api_url"
	PodResetRetryLimit = 5
	PodRetryInterval = 60
)

var (
	// Recorder to capture events
	Recorder = metric.PrometheusRecorder{}
)

// IdlerAPI defines the REST endpoints of the Idler
type IdlerAPI interface {
	// Idle triggers an idling of the Jenkins service running in the namespace specified in the namespace
	// parameter of the request. A status code of 200 indicates success whereas 500 indicates failure.
	Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// UnIdle triggers an un-idling of the Jenkins service running in the namespace specified in the namespace
	// parameter of the request. A status code of 200 indicates success whereas 500 indicates failure.
	UnIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// IsIdle returns an status struct indicating whether the Jenkins service in the namespace specified in the
	// namespace parameter of the request is currently idle or not.
	// If an error occurs a response with the HTTP status 500 is returned.
	IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// Status returns an statusResponse struct indicating the state of the
	// Jenkins service in the namespace specified in the namespace parameter
	// of the request.
	// If an error occurs a response with the HTTP status 400 or 500 is returned.
	Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// Info writes a JSON representation of internal state of the specified namespace to the response writer.
	Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// ClusterDNSView writes a JSON representation of the current cluster state to the response writer.
	ClusterDNSView(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// ResetNSPods deletes a pod and starts a new one
	Reset(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type idler struct {
	userIdlers      *openshift.UserIdlerMap
	clusterView     cluster.View
	openShiftClient client.OpenShiftClient
	controller      openshift.Controller
	tenantService   tenant.Service
}

type status struct {
	IsIdle bool `json:"is_idle"`
}

// NewIdlerAPI creates a new instance of IdlerAPI.
func NewIdlerAPI(userIdlers *openshift.UserIdlerMap, clusterView cluster.View, ts tenant.Service) IdlerAPI {
	// Initialize metrics
	Recorder.Initialize()
	return &idler{
		userIdlers:      userIdlers,
		clusterView:     clusterView,
		openShiftClient: client.NewOpenShift(),
		tenantService:   ts,
	}
}

func (api *idler) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	openShiftAPI, openShiftBearerToken, err := api.getURLAndToken(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	for _, service := range pidler.JenkinsServices {
		startTime := time.Now()
		err = api.openShiftClient.Idle(openShiftAPI, openShiftBearerToken, ps.ByName("namespace"), service)
		elapsedTime := time.Since(startTime).Seconds()

		if err != nil {
			Recorder.RecordReqDuration(service, "Idle", http.StatusInternalServerError, elapsedTime)
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		Recorder.RecordReqDuration(service, "Idle", http.StatusOK, elapsedTime)
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) UnIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	openshiftURL, openshiftToken, err := api.getURLAndToken(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	ns := strings.TrimSpace(ps.ByName("namespace"))
	if ns == "" {
		err = errors.New("Missing mandatory param namespace")
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	// may be jenkins is already running and in that case we don't have to do unidle it
	running, err := api.isJenkinsUnIdled(openshiftURL, openshiftToken, ns)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	} else if running {
		log.Infof("Jenkins is already starting/running on %s", ns)
		w.WriteHeader(http.StatusOK)
		return
	}

	// now that jenkins isn't running we need to check if the cluster has reached
	// its maximum capacity
	clusterFull, err := api.tenantService.HasReachedMaxCapacity(openshiftURL, ns)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	} else if clusterFull {
		err := fmt.Errorf("Maximum Resource limit reached on %s for %s", openshiftURL, ns)
		respondWithError(w, http.StatusServiceUnavailable, err)
		return
	}

	for _, service := range pidler.JenkinsServices {
		startTime := time.Now()
		err = api.openShiftClient.UnIdle(openshiftURL, openshiftToken, ns, service)
		elapsedTime := time.Since(startTime).Seconds()
		if err != nil {
			Recorder.RecordReqDuration(service, "UnIdle", http.StatusInternalServerError, elapsedTime)
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}
		// tries best to undle the pod
		go resetServicePods(api.openShiftClient, openshiftURL, openshiftToken, ns, service, 0);
		Recorder.RecordReqDuration(service, "UnIdle", http.StatusOK, elapsedTime)
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	openShiftAPI, openShiftBearerToken, err := api.getURLAndToken(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	state, err := api.openShiftClient.State(openShiftAPI, openShiftBearerToken, ps.ByName("namespace"), "jenkins")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	s := status{}
	s.IsIdle = state < model.PodRunning
	writeResponse(w, http.StatusOK, s)
}

func (api *idler) Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	response := &statusResponse{}

	openshiftURL, openshiftToken, err := api.getURLAndToken(r)
	if err != nil {
		response.AppendError(tokenFetchFailed, "failed to obtain openshift token: "+err.Error())
		writeResponse(w, http.StatusBadRequest, *response)
		return
	}

	state, err := api.openShiftClient.State(
		openshiftURL, openshiftToken,
		ps.ByName("namespace"),
		"jenkins",
	)
	if err != nil {
		response.AppendError(openShiftClientError, "openshift client error: "+err.Error())
		writeResponse(w, http.StatusInternalServerError, *response)
		return
	}

	response.SetState(state)
	writeResponse(w, http.StatusOK, *response)
}

func (api *idler) Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	namespace := ps.ByName("namespace")

	userIdler, ok := api.userIdlers.Load(namespace)
	if ok {
		writeResponse(w, http.StatusOK, userIdler.GetUser())
	} else {
		respondWithError(w, http.StatusNotFound, fmt.Errorf("Could not find queried namespace"))
	}
}

func (api *idler) ClusterDNSView(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	writeResponse(w, http.StatusOK, api.clusterView.GetDNSView())
}

func (api *idler) Reset(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	logger := log.WithFields(log.Fields{"component": "api", "function": "ResetNSPods"})

	openShiftAPI, openShiftBearerToken, err := api.getURLAndToken(r)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	err = api.openShiftClient.ResetNSPods(openShiftAPI, openShiftBearerToken, ps.ByName("namespace"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) getURLAndToken(r *http.Request) (string, string, error) {
	var openShiftAPIURL string
	values, ok := r.URL.Query()[OpenShiftAPIParam]
	if !ok || len(values) < 1 {
		return "", "", fmt.Errorf("OpenShift API URL needs to be specified")
	}

	openShiftAPIURL = values[0]
	bearerToken, ok := api.clusterView.GetToken(openShiftAPIURL)
	if ok {
		return openShiftAPIURL, bearerToken, nil
	}
	return "", "", fmt.Errorf("Unknown or invalid OpenShift API URL: %s", openShiftAPIURL)
}

func (api idler) isJenkinsUnIdled(openshiftURL, openshiftToken, namespace string) (bool, error) {
	state, err := api.openShiftClient.State(openshiftURL, openshiftToken, namespace, "jenkins")
	if err != nil {
		return false, err
	}

	status := state == model.PodStarting || state == model.PodRunning
	return status, nil
}

func respondWithError(w http.ResponseWriter, status int, err error) {
	log.Error(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
}

type responseError struct {
	Code        errorCode `json:"code"`
	Description string    `json:"description"`
}

type jenkinsInfo struct {
	State string `json:"state"`
}

type statusResponse struct {
	Data   *jenkinsInfo    `json:"data,omitempty"`
	Errors []responseError `json:"errors,omitempty"`
}

// ErrorCode is an integer that clients to can use to compare errors
type errorCode uint32

const (
	tokenFetchFailed     errorCode = 1
	openShiftClientError errorCode = 2
)

func (s *statusResponse) AppendError(code errorCode, description string) *statusResponse {
	s.Errors = append(s.Errors, responseError{
		Code:        code,
		Description: description,
	})
	return s
}

func (s *statusResponse) SetState(state model.PodState) *statusResponse {
	s.Data = &jenkinsInfo{State: state.String()}
	return s
}

type any interface{}

func writeResponse(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Could not serialize the response: %s", err))
		return
	}
}

/*could be better to move it to some other file*/
func resetServicePods(client client.OpenShiftClient, url string, token string, ns string, service string, retry int) bool {

	initiating := false
	resetOccurred := false

	if retry >= PodResetRetryLimit {
		log.Errorf("failed to reset the %s service in namespace %s", service, ns)
		return false
	}

	time.Sleep(PodRetryInterval * time.Second)

	states, err := client.PodState(url, token, ns, service);
	if err != nil {
		log.Error(err)
		return false
	}

	for n, s := range states {
		if podInitiating(s) {
			initiating = true;
			break;
		}

		if (failedToInitiatePod(s)) {
			resetOccurred = true
			error := client.ResetPod(url, token, ns, n)
			if error != nil {
				log.Warningf("failed to delete pod %s", n)
			}
			break;
		}
	}

	if initiating {
		return resetServicePods(client, url, token, ns, service, retry)
	} else if resetOccurred {
		retry++
		return resetServicePods(client, url, token, ns, service, retry)
	} else {
		return true
	}
}

func podInitiating(status v1.PodStatus) bool {
	return status.Phase == "Pending" && initiatingContainers(status)
}

func initiatingContainers(status v1.PodStatus) bool {

	for _, cs := range status.ContainerStatuses {
		if cs.State.Waiting != nil || cs.State.Running == nil {
			return true
		}
 	}

	for _, ics := range status.InitContainerStatuses {
		if ics.State.Waiting != nil || ics.State.Running == nil {
			return true
		}
	}

	return false
}

func failedToInitiatePod(status v1.PodStatus) bool {
	return status.Phase == "Failed" || status.Phase == "Pending" && containersFailed(status)
}

func containersFailed(status v1.PodStatus) bool {

	for _, cs := range status.ContainerStatuses {
		currentState := cs.State
		lastState    := cs.LastTerminationState

		if currentState.Terminated != nil && currentState.Terminated.Reason == "Error" && cs.RestartCount >= 1 ||
			lastState.Terminated != nil && lastState.Terminated.Reason == "Error" && cs.RestartCount >= 1 {
			return true
		}
	}

	for _, ics := range status.InitContainerStatuses {
		currentState := ics.State
		lastState    := ics.LastTerminationState

		if currentState.Terminated != nil && currentState.Terminated.Reason == "Error" && ics.RestartCount >= 1 ||
			lastState.Terminated != nil && lastState.Terminated.Reason == "Error" && ics.RestartCount >= 1 {
			return true
		}
	}

	return false
}