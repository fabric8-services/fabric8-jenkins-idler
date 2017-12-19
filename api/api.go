package api

import (
	"encoding/json"
	"fmt"
	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	oc "github.com/fabric8-services/fabric8-jenkins-idler/openshiftcontroller"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type IdlerAPI struct {
	OCli *ic.OpenShift
	OC   *oc.OpenShiftController
}

func NewAPI(o *ic.OpenShift, oc *oc.OpenShiftController) IdlerAPI {
	return IdlerAPI{
		OCli: o,
		OC:   oc,
	}
}

func (api *IdlerAPI) Builds(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
		fmt.Fprintf(w, "{'error': 'Could not serialize users'}")
	}

	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprintf(w, "{'error': 'Could not serialize users'}")
	}
}

func (api *IdlerAPI) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := api.OCli.Idle(ps.ByName("namespace"), "jenkins")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type Status struct {
	IsIdle bool `json:"is_idle"`
}

func (api *IdlerAPI) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	state, err := api.OCli.IsIdle(ps.ByName("namespace"), "jenkins")
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err)))
		return
	}

	s := Status{}
	s.IsIdle = state < ic.JenkinsRunning
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s)
}

func (api *IdlerAPI) GetRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
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
