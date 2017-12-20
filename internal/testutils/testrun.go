package testutils

import (
	"net/http"

	iClients "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshiftcontroller"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/common"
	pClients "github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

const (
	//How many times to retry to unidle before giving up
	unidleRetry = 15
)

func Run() {
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}

	tenantData, err := ioutil.ReadFile("internal/testutils/testdata/tenant.json")
	if err != nil {
		log.Fatal(err)
	}
	ts := common.MockServer(tenantData)
	defer ts.Close()

	o := iClients.NewOpenShift(config.GetOpenShiftURL(), config.GetOpenShiftToken())
	t := pClients.NewTenant(ts.URL, "xxx")

	oc := openshiftcontroller.NewOpenShiftController(o, t, config.GetConcurrentGroups(),
		config.GetIdleAfter(), config.GetFilteredNamespaces(), config.GetProxyURL(), unidleRetry, config.GetUseWatch())

	go oc.Run(0)

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

	//Start Idler API
	http.ListenAndServe(":8080", router)
}
