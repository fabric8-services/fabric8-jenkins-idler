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
	varUseWatch                        = "use.watch"
	varTenantURL                       = "f8tenant.api.url" 
	varAuthToken                       = "auth.token"
	varToggleURL                       = "toggle.api.url"

	varLocalDevEnv                     = "local.dev.env"
)

// Data encapsulates the Viper configuration object which stores the configuration data in-memory.
type Data struct {
	v *viper.Viper
}

// NewData creates a configuration reader object
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
	c.v.SetDefault(varUseWatch, true)
	c.v.SetDefault(varToggleURL, "http://f8toggles/api")
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

	if len(c.GetToggleURL()) == 0 {
		missingParam = true
		log.Error("You need to provide a Toggle Service URL in JC_TOGGLE_API_URL environment variable")
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
	fn := strings.Split(c.v.GetString(varFilteredNamespaces), ":")
	if len(fn) == 1 && len(fn[0]) == 0 {
		return []string{}
	}

	return fn
}

// GetConcurrentGroups returns the number of concurrent groups that shoul run as set via default, config file, or environment variable
func (c *Data) GetConcurrentGroups() int {
	return c.v.GetInt(varConcurrentGroups)
}

// GetIdleAfter returns the number of minutes before Jenkins is idled as set via default, config file, or environment variable
func (c *Data) GetIdleAfter() int {
	return c.v.GetInt(varIdleAfter)
}

// GetUseWatch returns if idler should use watch instead of poll as set via default, config file, or environment variable
func (c *Data) GetUseWatch() bool {
	return c.v.GetBool(varUseWatch)
}

// GetLocalDevEnv returns if it is local development env as set via default, config file, or environment variable
func (c *Data) GetLocalDevEnv() bool {
	return c.v.GetBool(varLocalDevEnv)
}

// GetTenantURL returns the F8 Tenant API URL as set via default, config file, or environment variable
func (c *Data) GetTenantURL() string {
	return c.v.GetString(varTenantURL)
}

// GetAuthToken returns the Auth token as set via default, config file, or environment variable
func (c *Data) GetAuthToken() string {
	return c.v.GetString(varAuthToken)
}

// GetToggleURL returns the Toggle Service URL as set via default, config file, or environment variable
func (c *Data) GetToggleURL() string {
	return c.v.GetString(varToggleURL)
}