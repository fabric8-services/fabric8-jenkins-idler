package tenant

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

// Service the interface for the cluster service
type Service interface {
	GetTenantInfoByNamespace(apiURL string, ns string) (InfoList, error)
}

// Tenant is a simple client for the fabric8-tenant service.
// The idea is to make this use a Goa client at this point. See issue #105.
type tenantService struct {
	tenantServiceURL string
	authToken        string
}

// NewTenantService returns an instance implementing Service.
func NewTenantService(tenantServiceURL string, authToken string) Service {
	return &tenantService{
		authToken:        authToken,
		tenantServiceURL: tenantServiceURL,
	}
}

// GetTenantInfoByNamespace gets you InfoList of a tenant given a namespace and api url.
func (t tenantService) GetTenantInfoByNamespace(apiURL string, ns string) (InfoList, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/tenants", t.tenantServiceURL), nil)
	if err != nil {
		return InfoList{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.authToken))

	q := req.URL.Query()
	q.Add("master_url", util.EnsureSuffix(apiURL, "/"))
	q.Add("namespace", ns)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return InfoList{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return InfoList{}, err
	}

	tenantInfo := InfoList{}
	err = json.Unmarshal(body, &tenantInfo)
	if err != nil {
		return InfoList{}, err
	}

	return tenantInfo, nil
}
