package mock

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// IdlerAPI defines the REST endpoints of the Idler
type IdlerAPI struct {
}

// Idle triggers an idling of the Jenkins service running in the namespace specified in the namespace
// parameter of the request. A status code of 200 indicates success whereas 500 indicates failure.
func (i *IdlerAPI) Idle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("Idle"))
	w.WriteHeader(http.StatusOK)
}

// UnIdle triggers an un-idling of the Jenkins service running in the namespace specified in the namespace
// parameter of the request. A status code of 200 indicates success whereas 500 indicates failure.
func (i *IdlerAPI) UnIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("UnIdle"))
	w.WriteHeader(http.StatusOK)
}

// IsIdle returns an status struct indicating whether the Jenkins service in the namespace specified in the
// namespace parameter of the request is currently idle or not.
// If an error occurs a response with the HTTP status 500 is returned.
func (i *IdlerAPI) IsIdle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("IsIdle"))
	w.WriteHeader(http.StatusOK)
}

// Status returns an StatusResponse struct indicating whether the Jenkins service
// in the namespace specified in the namespace parameter of the request is
// idle, starting or running.
// If an error occurs a response with the HTTP status 400 or 500 is returned.
func (i *IdlerAPI) Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("Status"))
	w.WriteHeader(http.StatusOK)
}

// Reset mock resets pods
func (i *IdlerAPI) Reset(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Reset"))
}

// ClusterDNSView writes a JSON representation of the current cluster state to the response writer.
func (i *IdlerAPI) ClusterDNSView(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	w.Write([]byte("GetClusterDNSView"))
	w.WriteHeader(http.StatusOK)
}

//SetUserIdlerStatus sets the user status
func (i *IdlerAPI) SetUserIdlerStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	_, err := w.Write([]byte("SetUserIdlerStatus"))
	if err != nil {
		log.Error(err)
	}
	w.WriteHeader(http.StatusOK)
}

//GetDisabledUserIdlers set the user status
func (i *IdlerAPI) GetDisabledUserIdlers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, err := w.Write([]byte("GetDisabledUserIdlers")); err != nil {
		log.Error(err)
	}
	w.WriteHeader(http.StatusOK)
}
