package testutils

import (
	"net/http"

	"io/ioutil"

	iClients "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshiftcontroller"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/common"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	pClients "github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
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

	//Create Toggle (Unleash) Service client
	features, err := toggles.NewUnleashToggle("http://unleash.herokuapp.com/api/")
	if err != nil {
		log.Fatal(err)
	}

	oc := openshiftcontroller.NewOpenShiftController(
		&o,
		&t,
		config.GetIdleAfter(),
		config.GetFilteredNamespaces(),
		config.GetProxyURL(),
		unidleRetry,
		features,
	)

	go oc.Run()

	//Create router for Idler API
	router := httprouter.New()
	api := api.NewIdlerAPI(&o, oc)

	router.GET("/iapi/idler/builds/", api.User)
	router.GET("/iapi/idler/builds/:namespace", api.User)
	router.GET("/iapi/idler/builds/:namespace/", api.User)
	router.GET("/iapi/idler/idle/:namespace", api.Idle)
	router.GET("/iapi/idler/idle/:namespace/", api.Idle)
	router.GET("/iapi/idler/isidle/:namespace", api.IsIdle)
	router.GET("/iapi/idler/isidle/:namespace/", api.IsIdle)
	router.GET("/iapi/idler/route/:namespace", api.GetRoute)
	router.GET("/iapi/idler/route/:namespace/", api.GetRoute)

	//Start Idler API
	http.ListenAndServe(":8080", router)
}
