package configuration

import (
	"os"
	"reflect"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

func TestConfig_GetDebugMode(t *testing.T) {
	os.Setenv(DebugMode, "false")
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "Config Debug",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetDebugMode(); got != tt.want {
				t.Errorf("Config.GetDebugMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetProxyURL(t *testing.T) {
	os.Setenv(ProxyURL, "https://proxy.openshift.io")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test Get Proxy",
			want: "https://proxy.openshift.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetProxyURL(); got != tt.want {
				t.Errorf("Config.GetProxyURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetTenantURL(t *testing.T) {
	os.Setenv(TenantURL, "https://tenent.openshift.io")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test GetTenant",
			want: "https://tenent.openshift.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetTenantURL(); got != tt.want {
				t.Errorf("Config.GetTenantURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetToggleURL(t *testing.T) {
	os.Setenv(ToggleURL, "https://toggle.openshift.io")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test GetToggle URL",
			want: "https://toggle.openshift.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetToggleURL(); got != tt.want {
				t.Errorf("Config.GetToggleURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetAuthURL(t *testing.T) {
	os.Setenv(AuthURL, "https://auth.openshift.io")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test GetAuthURL",
			want: "https://auth.openshift.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetAuthURL(); got != tt.want {
				t.Errorf("Config.GetAuthURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetServiceAccountID(t *testing.T) {
	os.Setenv(ServiceAccountID, "1234567")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test GetServiceAccountID",
			want: "1234567",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetServiceAccountID(); got != tt.want {
				t.Errorf("Config.GetServiceAccountID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetServiceAccountSecret(t *testing.T) {
	os.Setenv(ServiceAccountSecret, "secret")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test GetServiceAccountSecret",
			want: "secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetServiceAccountSecret(); got != tt.want {
				t.Errorf("Config.GetServiceAccountSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetAuthTokenKey(t *testing.T) {
	os.Setenv(AuthTokenKey, "tokenkey")
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test GetAuthTokenKey",
			want: "tokenkey",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetAuthTokenKey(); got != tt.want {
				t.Errorf("Config.GetAuthTokenKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetAuthGrantType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test GetAuthGrantType",
			want: "client_credentials",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetAuthGrantType(); got != tt.want {
				t.Errorf("Config.GetAuthGrantType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetIdleAfter(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "Test GetIdleAfter",
			want: defaultIdleAfter,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetIdleAfter(); got != tt.want {
				t.Errorf("Config.GetIdleAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetIdleLongBuild(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "Test GetIdleLongBuild",
			want: defaultIdleLongBuild,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetIdleLongBuild(); got != tt.want {
				t.Errorf("Config.GetIdleLongBuild() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetMaxRetries(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "Test GetMaxRetries",
			want: defaultMaxRetries,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetMaxRetries(); got != tt.want {
				t.Errorf("Config.GetMaxRetries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetMaxRetriesQuietInterval(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "Test GetMaxRetriesQuietInterval",
			want: defaultMaxRetriesQuietInterval,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetMaxRetriesQuietInterval(); got != tt.want {
				t.Errorf("Config.GetMaxRetriesQuietInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetCheckInterval(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "Test GetCheckInterval",
			want: defaultCheckInterval,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetCheckInterval(); got != tt.want {
				t.Errorf("Config.GetCheckInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetFixedUuids(t *testing.T) {
	os.Setenv(FixedUuids, "uuid1,uuid2,uuid3")
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "Test GetFixedUuids",
			want: []string{"uuid1", "uuid2", "uuid3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.GetFixedUuids(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config.GetFixedUuids() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_Verify(t *testing.T) {
	os.Clearenv()
	os.Setenv(AuthTokenKey, "tokenkey")
	os.Setenv(ServiceAccountSecret, "secret")
	os.Setenv(ServiceAccountID, "1234567")
	os.Setenv(AuthURL, "https://auth.openshift.io")
	os.Setenv(ToggleURL, "https://toggle.openshift.io")
	os.Setenv(TenantURL, "https://tenent.openshift.io")
	os.Setenv(ProxyURL, "https://proxy.openshift.io")

	tests := []struct {
		name string
		want util.MultiError
	}{
		{
			name: "Test Config Verify",
			want: util.MultiError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := New("")
			if got := c.Verify(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config.Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		configFilePath string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name:    "Test Config New",
			args:    args{"test.yaml"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.configFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
