package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
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

	// User writes a JSON representation of the User struct to the HTTP response.
	// If no namespace parameter is specified all Users are included into the response. If the namespace
	// parameter is set only the user with the specified namespace gets added to the response.
	//
	// NOTE: This endpoint is for debugging purposes and will be removed at some stage.
	User(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type idler struct {
	openShiftClient client.OpenShiftClient
	controller      openshift.ControllerI
}

type status struct {
	IsIdle bool `json:"is_idle"`
}

// NewIdlerAPI creates a new instance of IdlerAPI.
func NewIdlerAPI(openShiftClient client.OpenShiftClient, controller openshift.ControllerI) IdlerAPI {
	return &idler{
		openShiftClient: openShiftClient,
		controller:      controller,
	}
}

func (api *idler) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := api.openShiftClient.Idle(ps.ByName("namespace"), "jenkins")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) UnIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := api.openShiftClient.UnIdle(ps.ByName("namespace"), "jenkins")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	state, err := api.openShiftClient.IsIdle(ps.ByName("namespace"), "jenkins")
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

func (api *idler) User(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	ns := ps.ByName("namespace")

	err := json.NewEncoder(w).Encode(api.controller.GetUser(ns))

	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprint(w, "{'error': 'Could not serialize users'}")
	}

	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprint(w, "{'error': 'Could not serialize users'}")
	}
}
