// You can edit this code!
// Click here and start typing.
package main

import (
	"strings"

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
	token := v.GetString("openshift.api.token")
	if len(token) == 0 {
		missingParam = true
		log.Error("You need to provide an OpenShift access token in JC_OPENSHIFT_API_TOKEN environment variable")
	}

	if missingParam {
		log.Panic("A value for envinronment variable is missing")
	}
	namespaceArg := v.GetString("openshift.namespace")
	namespaces := strings.Split(namespaceArg, ":")


	oc := openshiftcontroller.NewOpenShiftController(apiURL, token)

	oc.Run(namespaces)
	
	
}