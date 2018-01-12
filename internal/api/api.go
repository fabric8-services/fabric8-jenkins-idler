package api

import (
	"encoding/json"
	"fmt"
	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	oc "github.com/fabric8-services/fabric8-jenkins-idler/internal/openshiftcontroller"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// TODO - Eventually we might want to use goa to define the API and potentially generate a REST client (HF)
// IdlerAPI defines the REST endpoints of the Idler
type IdlerAPI interface {
	// Idle triggers an idling of the Jenkins service running in the namespace specified in the namespace
	// parameter of the request. A status code of 200 indicates success whereas 500 indicates failure.
	Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// IsIdle returns an status struct indicating whether the Jenkins service in the namespace specified in the
	// namespace parameter of the request is currently idle or not.
	// If an error occurs a response with the HTTP status 500 is returned.
	IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params)

	// GetRoute returns an route struct containing information about the route of the Jenkins service in the
	// namespace specified in the namespace parameter of the request.
	// If an error occurs a response with the HTTP status 500 is returned.
	GetRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params)

	// User writes a JSON representation of the User struct to the HTTP response.
	// If no namespace parameter is specified all Users are included into the response. If the namespace
	// parameter is set only the user with the specified namespace gets added to the response.
	//
	// NOTE: This endpoint is for debugging purposes and will be removed at some stage.
	User(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
}

type idler struct {
	OCli *ic.OpenShift
	OC   *oc.OpenShiftController
}

type status struct {
	IsIdle bool `json:"is_idle"`
}

// NewIdlerAPI creates a new instance of IdlerAPI.
func NewIdlerAPI(o *ic.OpenShift, oc *oc.OpenShiftController) IdlerAPI {
	return &idler{
		OCli: o,
		OC:   oc,
	}
}

func (api *idler) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := api.OCli.Idle(ps.ByName("namespace"), "jenkins")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *idler) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	state, err := api.OCli.IsIdle(ps.ByName("namespace"), "jenkins")
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	s := status{}
	s.IsIdle = state < ic.JenkinsRunning
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s)
}

func (api *idler) GetRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	namespace := ps.ByName("namespace")
	w.Header().Set("Content-Type", "application/json")

	r, tls, err := api.OCli.GetRoute(namespace, "jenkins")
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	type route struct {
		Service string `json:"service"`
		Route   string `json:"route"`
		TLS     bool   `json:"tls"`
	}

	rt := route{
		Route:   r,
		Service: "jenkins",
		TLS:     tls,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rt)
}

func (api *idler) User(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var err error
	w.Header().Set("Content-Type", "application/json")
	ns := ps.ByName("namespace")
	if len(ns) > 0 {
		err = json.NewEncoder(w).Encode(api.OC.Users[ns])
	} else {
		err = json.NewEncoder(w).Encode(api.OC.Users)
	}

	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprint(w, "{'error': 'Could not serialize users'}")
	}

	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprint(w, "{'error': 'Could not serialize users'}")
	}
}
