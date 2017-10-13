// You can edit this code!
// Click here and start typing.
package main

import (
	"time"
	"strings"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-idler/openshiftcontroller"

	viper "github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
)

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
	token := v.GetString("openshift.api.token")
	if len(token) == 0 {
		missingParam = true
		log.Error("You need to provide an OpenShift access token in JC_OPENSHIFT_API_TOKEN environment variable")
	}

	nGroups := v.GetInt("concurrent.groups")
	if nGroups == 0 {
		nGroups = 1
	}

	if missingParam {
		log.Panic("A value for envinronment variable is missing")
	}
	//namespaceArg := v.GetString("openshift.namespace")
	//namespaces := strings.Split(namespaceArg, ":")


	oc := openshiftcontroller.NewOpenShiftController(apiURL, token, nGroups)

	//FIXME!
	http.HandleFunc("/builds", oc.ServeJenkinsStates)
	
	for gn, g := range oc.Groups {
		go oc.Run(g, gn)
		time.Sleep(2*time.Second)
	}
	
	http.ListenAndServe(":8080", nil)
}