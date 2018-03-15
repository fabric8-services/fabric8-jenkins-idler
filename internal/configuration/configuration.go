package configuration

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

// Configuration defines the configuration options of the Idler.
type Configuration interface {
	// GetProxyURL returns the Jenkins Proxy API URL.
	GetProxyURL() string

	// GetTenantURL returns the F8 Tenant API URL.
	GetTenantURL() string

	// GetToggleURL returns the Toggle Service URL.
	GetToggleURL() string

	// GetIdleAfter returns the number of minutes before Jenkins is idled.
	GetIdleAfter() int

	// GetMaxRetries returns the maximum number of retries to idle resp. un-idle the Jenkins service.
	GetMaxRetries() int

	// GetMaxRetriesQuietInterval returns the number of minutes no retry occurs after the maximum retry count is reached.
	GetMaxRetriesQuietInterval() int

	// GetCheckInterval returns the number of minutes after which a regular idle check occurs.
	GetCheckInterval() int

	// GetDebugMode returns if debug mode should be enabled.
	GetDebugMode() bool

	// GetFixedUuids returns a slice of fixed user uuids. If set, a custom Features implementation is instantiated
	// which only enabled the idler feature for the specified list of users. This is mainly used for local dev only.
	GetFixedUuids() []string

	// GetAuthURL returns the Auth API URL as set via default, config file, or environment variable
	GetAuthURL() string

	// GetServiceAccountID returns the service account id for the Auth service. Used to identify the Idler to the Auth service
	GetServiceAccountID() string

	// GetServiceAccountSecret returns the service account secret. Used to authenticate the Idler to the Auth service.
	GetServiceAccountSecret() string

	// GetAuthTokenKey returns the key to decrypt OpenShift API tokens obtained via the Cluster API.
	GetAuthTokenKey() string

	// GetAuthGrantType returns the fabric8-auth Grant type used while retrieving
	// user account token
	GetAuthGrantType() string

	// Verify validates the configuration and returns an error in case the configuration is missing required settings
	// or contains invalid settings. If the configuration is correct nil is returned.
	Verify() util.MultiError

	// String returns the current configuration as string.
	String() string
}
