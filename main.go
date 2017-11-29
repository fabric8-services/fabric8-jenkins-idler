// You can edit this code!
// Click here and start typing.
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
	unidleRetry = 15
)

func init() {
  log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	config.Verify()

	o := iClients.NewOpenShift(config.GetOpenShiftURL(), config.GetOpenShiftToken())

	oc := openshiftcontroller.NewOpenShiftController(o, config.GetConcurrentGroups(),
										config.GetIdleAfter(), config.GetFilteredNamespaces(), config.GetProxyURL(), unidleRetry)

	//FIXME!

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
	
	for gn, _ := range oc.Groups {
		go oc.Run(gn, config.GetUseWatch())
	}

	if !config.GetUseWatch() {
		go func() {
			for {
				oc.DownloadProjects()
				time.Sleep(10*time.Minute)
			}
		}()
	}
	
	go http.ListenAndServe(":8080", router)
	http.ListenAndServe(":4000", nil)
}