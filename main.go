package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/configuration"
	"time"
	"net/http"
	_ "net/http/pprof"

	"github.com/fabric8-services/fabric8-jenkins-idler/openshiftcontroller"
	"github.com/fabric8-services/fabric8-jenkins-idler/api"

	iClients "github.com/fabric8-services/fabric8-jenkins-idler/clients"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

const (
	//How many times to retry to unidle before giving up
	unidleRetry = 15
)

func init() {
  log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	//Init configuration
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	//Verify if we have all the info
	config.Verify()

	//Create OpenShift client
	o := iClients.NewOpenShift(config.GetOpenShiftURL(), config.GetOpenShiftToken())

	//Create Idler controller
	oc := openshiftcontroller.NewOpenShiftController(o, config.GetConcurrentGroups(),
										config.GetIdleAfter(), config.GetFilteredNamespaces(), config.GetProxyURL(), unidleRetry, config.GetUseWatch())

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
	for gn, _ := range oc.Groups {
		go oc.Run(gn)
	}

	//If we do not use websocket to get events from OpenShift, we need to update list of projects regularly (to spot new users)
	if !config.GetUseWatch() {
		go func() {
			for {
				oc.DownloadProjects()
				time.Sleep(10*time.Minute)
			}
		}()
	}
	
	//Start Idler API
	http.ListenAndServe(":8080", router)
}