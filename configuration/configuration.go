package configuration

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	varOpenShiftToken                  = "openshift.api.token"
	varOpenShiftURL                    = "openshift.api.url"
	varProxyURL                        = "jenkins.proxy.api.url"
	varConcurrentGroups                = "concurrent.groups"
	varIdleAfter                       = "idle.after"	
	varFilteredNamespaces              = "filter.namespaces"

	varLocalDevEnv                     = "local.dev.env"
)

// Data encapsulates the Viper configuration object which stores the configuration data in-memory.
type Data struct {
	v *viper.Viper
}

// NewData creates a configuration reader object using a configurable configuration file path
func NewData() (*Data, error) {
	c := Data{
		v: viper.New(),
	}
	c.v.SetEnvPrefix("JC")
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()

	return &c, nil
}

func (c *Data) setConfigDefaults() {
	//---------
	// Postgres
	//---------
	c.v.SetTypeByDefaultValue(true)
	c.v.SetDefault(varIdleAfter, 30)
	c.v.SetDefault(varConcurrentGroups, 1)
	c.v.SetDefault(varProxyURL, "http://localhost:9091")
}

func (c *Data) Verify() {
	missingParam := false
	apiURL := c.GetOpenShiftURL()
	if len(apiURL) == 0 {
		missingParam = true
		log.Error("You need to provide URL to OpenShift API endpoint in JC_OPENSHIFT_API_URL environment variable")
	}

	if apiURL[len(apiURL)-1] == '/' {
		apiURL = apiURL[:len(apiURL)-2]
	}

	proxyURL := c.GetProxyURL()
	if len(proxyURL) > 0 {
		if !strings.HasPrefix(proxyURL, "https://") && !strings.HasPrefix(proxyURL, "http://") {
			missingParam = true
			log.Error("Please provide a protocol - http(s) - for proxy url: ", proxyURL)
		}
	}

	token := c.GetOpenShiftToken()
	if len(token) == 0 {
		missingParam = true
		log.Error("You need to provide an OpenShift access token in JC_OPENSHIFT_API_TOKEN environment variable")
	}

	nGroups := c.GetConcurrentGroups()
	if nGroups == 0 {
		nGroups = 1
	}

	idleAfter := c.GetIdleAfter()
	if idleAfter == 0 {
		idleAfter = 10
	}

	if missingParam {
		log.Fatal("A value for envinronment variable is missing or wrong")
	}
}

// GetOpenShiftToken returns the OpenShift token as set via default, config file, or environment variable
func (c *Data) GetOpenShiftToken() string {
	return c.v.GetString(varOpenShiftToken)
}

// GetOpenShiftURL returns the OpenShift API url as set via default, config file, or environment variable
func (c *Data) GetOpenShiftURL() string {
	return c.v.GetString(varOpenShiftURL)
}

// GetProxyURL returns the Jenkins Proxy API URL as set via default, config file, or environment variable
func (c *Data) GetProxyURL() string {
	return c.v.GetString(varProxyURL)
}

// GetFilteredNamespaces returns the list of namespaces to handle as set via default, config file, or environment variable
func (c *Data) GetFilteredNamespaces() []string {
	return strings.Split(c.v.GetString(varFilteredNamespaces), ":")
}

// GetConcurrentGroups returns the number of concurrent groups that shoul run as set via default, config file, or environment variable
func (c *Data) GetConcurrentGroups() int {
	return c.v.GetInt(varConcurrentGroups)
}

// GetIdleAfter returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable
func (c *Data) GetIdleAfter() int {
	return c.v.GetInt(varIdleAfter)
}

// GetLocalDevEnv returns if it is local development env as set via default, config file, or environment variable
func (c *Data) GetLocalDevEnv() bool {
	return c.v.GetBool(varLocalDevEnv)
}