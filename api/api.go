package api

import (
	"fmt"
	"encoding/json"
	"net/http"
	"github.com/julienschmidt/httprouter"
	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	oc "github.com/fabric8-services/fabric8-jenkins-idler/openshiftcontroller"
	log "github.com/sirupsen/logrus"
)

type IdlerAPI struct {
	OCli *ic.OpenShift
	OC *oc.OpenShiftController
}

func NewAPI(o *ic.OpenShift, oc *oc.OpenShiftController) IdlerAPI {
	return IdlerAPI{
		OCli: o,
		OC: oc,
	}
}

func (api *IdlerAPI) Builds(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(api.OC.Users[ps.ByName("namespace")])
	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprintf(w, "{'error': 'Could not serialize users'}")
	}

	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprintf(w, "{'error': 'Could not serialize users'}")
	}
}

func (api *IdlerAPI) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := api.OCli.Idle(ps.ByName("namespace"), "jenkins")
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type Status struct {
	IsIdle bool `json:"is_idle"`
}

func (api *IdlerAPI) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	state, err := api.OCli.IsIdle(ps.ByName("namespace"), "jenkins")
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
	}

	s := Status{}
	s.IsIdle = state < ic.JenkinsStates["Running"]
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s)
}

func (api *IdlerAPI) GetRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	namespace := ps.ByName("namespace")
	r, err := api.OCli.GetRoute(namespace, "jenkins")
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	type route struct {
		Service string `json:"service"`
		Route string `json:"route"`
	}

	rt := route{
		Route: r,
		Service: "jenkins",
	}

	json.NewEncoder(w).Encode(rt)
	w.WriteHeader(http.StatusOK)
}