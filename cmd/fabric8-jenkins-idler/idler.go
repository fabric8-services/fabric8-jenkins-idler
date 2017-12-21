package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshiftcontroller"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"

	iClients "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	pClients "github.com/fabric8-services/fabric8-jenkins-proxy/clients"

	"github.com/julienschmidt/httprouter"
)

const (
	//How many times to retry to unidle before giving up
	unidleRetry = 15
	//How many times to wait for toggles service to become ready
	togglesReadyRetry = 10
)

type Idler struct {
	features toggles.Features
	config   *configuration.Data
}

func NewIdler(config *configuration.Data, features toggles.Features) *Idler {
	return &Idler{
		config:   config,
		features: features,
	}
}

func (idler *Idler) Run() {
	//Create OpenShift client
	o := iClients.NewOpenShift(idler.config.GetOpenShiftURL(), idler.config.GetOpenShiftToken())

	//Create Tenant client
	t := pClients.NewTenant(idler.config.GetTenantURL(), idler.config.GetAuthToken())

	//Create Idler controller
	oc := openshiftcontroller.NewOpenShiftController(
		o,
		t,
		idler.config.GetConcurrentGroups(),
		idler.config.GetIdleAfter(),
		idler.config.GetFilteredNamespaces(),
		idler.config.GetProxyURL(),
		unidleRetry,
		idler.config.GetUseWatch(),
		idler.features,
	)

	//Create router for Idler API
	router := httprouter.New()
	api := api.NewAPI(&o, oc)

	router.GET("/iapi/idler/builds/", api.Builds)
	router.GET("/iapi/idler/builds/:namespace", api.Builds)
	router.GET("/iapi/idler/builds/:namespace/", api.Builds)
	router.GET("/iapi/idler/idle/:namespace", api.Idle)
	router.GET("/iapi/idler/idle/:namespace/", api.Idle)
	router.GET("/iapi/idler/isidle/:namespace", api.IsIdle)
	router.GET("/iapi/idler/isidle/:namespace/", api.IsIdle)
	router.GET("/iapi/idler/route/:namespace", api.GetRoute)
	router.GET("/iapi/idler/route/:namespace/", api.GetRoute)

	//Spawn the main loop
	for gn := range oc.Groups {
		go oc.Run(gn)
	}

	//If we do not use websocket to get events from OpenShift, we need to update list of projects regularly (to spot new users)
	if !idler.config.GetUseWatch() {
		go func() {
			for {
				oc.DownloadProjects()
				time.Sleep(10 * time.Minute)
			}
		}()
	}

	//Start Idler API
	http.ListenAndServe(":8080", router)
}
