package main

import (
	"os"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
)

type testTenantService struct {
	IdentityID string
	BaseName   string
}

func (t testTenantService) GetTenantInfoByNamespace(apiURL string, ns string) (tenant.InfoList, error) {
	if ns == t.BaseName || ns == t.BaseName+"-jenkins" {
		return tenant.InfoList{
			Data: []tenant.InfoData{{
				ID: t.IdentityID,
				Attributes: tenant.Attributes{
					Namespaces: []tenant.Namespace{
						{
							ClusterURL: apiURL,
							Name:       ns,
							Type:       "user",
							ClusterCapacityExhausted: false,
						},
						{
							ClusterURL: apiURL,
							Name:       ns + "-jenkins",
							Type:       "jenkins",
							ClusterCapacityExhausted: false,
						},
					},
				},
			}},
		}, nil
	}
	return tenant.InfoList{}, nil
}

func (t testTenantService) HasReachedMaxCapacity(apiURL, ns string) (bool, error) {
	return false, nil
}

type testFeaturesService struct {
	IdentityID string
}

func (t testFeaturesService) IsIdlerEnabled(uid string) (bool, error) {
	if uid == t.IdentityID {
		return true, nil
	}
	return false, nil
}

func NewTestIdler(baseName, identityID, clusterURL, token string) *Idler {
	featuresService := testFeaturesService{
		IdentityID: identityID,
	}
	tenantService := testTenantService{
		IdentityID: identityID,
		BaseName:   baseName,
	}
	clusterView := cluster.NewView([]cluster.Cluster{
		cluster.Cluster{
			APIURL: clusterURL,
			AppDNS: "4e1e.starter-us-east-2a.openshiftapps.com",
			Token:  token,
		},
	})
	config, _ := configuration.NewConfiguration()

	return NewIdler(featuresService, tenantService, clusterView, config, baseName)
}

var (
	baseName   = "aslak-preview"
	identityID = "81d3c8d8-1aa2-49b4-a4c3-ffb41ed1f439"
	clusterURL = "https://api.starter-us-east-2a.openshift.com/"
	token      = "xxxxxxxx"
)

func main() {

	os.Setenv("JC_DEBUG_MODE", "true")
	os.Setenv("JC_IDLE_AFTER", "5")
	os.Setenv("JC_CHECK_INTERVAL", "5")
	os.Setenv("JC_MAX_RETRIES", "5")
	os.Setenv("JC_MAX_RETRIES_QUIET_INTERVAL", "5")

	idler := NewTestIdler(baseName, identityID, clusterURL, token)
	idler.Run()
}
