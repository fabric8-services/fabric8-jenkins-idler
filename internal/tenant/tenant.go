package tenant

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

// Tenant is a simple client for the fabric8-tenant service
// The idea is to make this use a Goa client at this point. See issue #105
type Tenant struct {
	tenantServiceURL string
	authToken        string
}

// NewTenant creates a new instance of type Tenant
func NewTenant(tenantServiceURL string, authToken string) Tenant {
	return Tenant{
		authToken:        authToken,
		tenantServiceURL: tenantServiceURL,
	}
}

// GetTenantInfoByNamespace get you InfoList of a tanent given a namespace and api url
func (t Tenant) GetTenantInfoByNamespace(apiURL string, ns string) (InfoList, error) {
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
