// You can edit this code!
// Click here and start typing.
package main

import (
	"strings"

	"github.com/vpavlin/jenkins-controller/openshiftcontroller"

	viper "github.com/spf13/viper"
)

func main() {

	v := viper.New()
	v.SetEnvPrefix("JC")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetTypeByDefaultValue(true)

	apiURL := v.GetString("openshift.api.url")
	token := v.GetString("openshift.api.token")
	namespace_arg := v.GetString("openshift.namespace")
	namespaces := strings.Split(namespace_arg, ":")


	oc := openshiftcontroller.NewOpenShiftController(apiURL, token)

	oc.Run(namespaces)
	
	
}