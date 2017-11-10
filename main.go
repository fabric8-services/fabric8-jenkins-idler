// You can edit this code!
// Click here and start typing.
package main

import (
	"time"
	"strings"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-idler/openshiftcontroller"
	"github.com/fabric8-services/fabric8-jenkins-idler/api"

	iClients "github.com/fabric8-services/fabric8-jenkins-idler/clients"

	"github.com/julienschmidt/httprouter"
	viper "github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
)

func init() {
  log.SetFormatter(&log.JSONFormatter{})
}

func main() {

	v := viper.New()
	v.SetEnvPrefix("JC")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetTypeByDefaultValue(true)

	missingParam := false
	apiURL := v.GetString("openshift.api.url")
	if len(apiURL) == 0 {
		missingParam = true
		log.Error("You need to provide URL to OpenShift API endpoint in JC_OPENSHIFT_API_URL environment variable")
	}

	if strings.HasPrefix(apiURL, "https://") {
		apiURL = apiURL[8:]
	}

	if apiURL[len(apiURL)-1] == '/' {
		apiURL = apiURL[:len(apiURL)-2]
	}

	proxyURL := v.GetString("jenkins.proxy.url")

	if len(proxyURL) > 0 {
		if !strings.HasPrefix(proxyURL, "https://") && !strings.HasPrefix(proxyURL, "http://") {
			missingParam = true
			log.Error("Please provide a protocol - http(s) - for proxy url: ", proxyURL)
		}

		if proxyURL[len(proxyURL)-1] == '/' {
			proxyURL = proxyURL[:len(proxyURL)-2]
		}
	}

	token := v.GetString("openshift.api.token")
	if len(token) == 0 {
		missingParam = true
		log.Error("You need to provide an OpenShift access token in JC_OPENSHIFT_API_TOKEN environment variable")
	}

	nGroups := v.GetInt("concurrent.groups")
	if nGroups == 0 {
		nGroups = 1
	}

	idleAfter := v.GetInt("idle.after")
	if idleAfter == 0 {
		idleAfter = 10
	}

	if missingParam {
		log.Fatal("A value for envinronment variable is missing or wrong")
	}
	namespaceArg := v.GetString("filter.namespaces")
	namespaces := strings.Split(namespaceArg, ":")

	o := iClients.NewOpenShift(apiURL, token)

	oc := openshiftcontroller.NewOpenShiftController(o, nGroups, idleAfter, namespaces, proxyURL)

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
		go oc.Run(gn)
	}

	go func() {
		oc.DownloadProjects()
		time.Sleep(1*time.Minute)
	}()
	
	http.ListenAndServe(":8080", router)
}