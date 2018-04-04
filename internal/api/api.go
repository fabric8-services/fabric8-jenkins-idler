package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/metric"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

const (
	// OpenShiftAPIParam is the parameter name under which the OpenShift cluster API URL is passed using
	// Idle, UnIdle and IsIdle.
	OpenShiftAPIParam = "openshift_api_url"
)

var (
	// JenkinsServices is an array of all the services getting idled or unidled
	// they go along the main build detection logic of jenkins and don't have
	// any specific scenarios.
	JenkinsServices = []string{"jenkins", "content-repository"}

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

	// Info writes a JSON representation of internal state of the specified namespace to the response writer.
	Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// ClusterDNSView writes a JSON representation of the current cluster state to the response writer.
	ClusterDNSView(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type idler struct {
	userIdlers      *openshift.UserIdlerMap
	clusterView     cluster.View
	openShiftClient client.OpenShiftClient
	controller      openshift.Controller
}

type status struct {
	IsIdle bool `json:"is_idle"`
}

// NewIdlerAPI creates a new instance of IdlerAPI.
func NewIdlerAPI(userIdlers *openshift.UserIdlerMap, clusterView cluster.View) IdlerAPI {
	// Initialize metrics
	Recorder.Initialize()
	return &idler{
		userIdlers:      userIdlers,
		clusterView:     clusterView,
		openShiftClient: client.NewOpenShift(),
	}
}

func (api *idler) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	openShiftAPI, openShiftBearerToken, err := api.getURLAndToken(r)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	for _, service := range JenkinsServices {
		startTime := time.Now()
		err = api.openShiftClient.Idle(openShiftAPI, openShiftBearerToken, ps.ByName("namespace"), service)
		elapsedTime := time.Since(startTime).Seconds()

		if err != nil {
			log.Error(err)
			Recorder.RecordReqDuration(service, "Idle", http.StatusInternalServerError, elapsedTime)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
			return
		}

		Recorder.RecordReqDuration(service, "Idle", http.StatusOK, elapsedTime)
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) UnIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	openShiftAPI, openShiftBearerToken, err := api.getURLAndToken(r)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	for _, service := range JenkinsServices {

		err = api.openShiftClient.UnIdle(openShiftAPI, openShiftBearerToken, ps.ByName("namespace"), service)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	openShiftAPI, openShiftBearerToken, err := api.getURLAndToken(r)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	state, err := api.openShiftClient.IsIdle(openShiftAPI, openShiftBearerToken, ps.ByName("namespace"), "jenkins")
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	s := status{}
	s.IsIdle = state < model.JenkinsRunning
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s)
}

func (api *idler) ClusterDNSView(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	clusterDNSView := api.clusterView.GetDNSView()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(clusterDNSView)
}

func (api *idler) Info(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	namespace := ps.ByName("namespace")

	userIdler, ok := api.userIdlers.Load(namespace)
	if ok {
		err := json.NewEncoder(w).Encode(userIdler.GetUser())

		if err != nil {
			log.Errorf("Could not serialize users: %s", err)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"Could not serialize users: %s\"}", err)))
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
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
	return "", "", fmt.Errorf("Unknown or invalid OpenShift API URL")
}
