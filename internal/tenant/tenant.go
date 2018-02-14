package tenant

import (
	"encoding/json"
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
	"io/ioutil"
	"net/http"
)

// Tenant is a simple client for the fabric8-tenant service
// The idea is to make this use a Goa client at this point. See issue #105
type Tenant struct {
	tenantServiceURL string
	authToken        string
}

func NewTenant(tenantServiceURL string, authToken string) Tenant {
	return Tenant{
		authToken:        authToken,
		tenantServiceURL: tenantServiceURL,
	}
}

func (t Tenant) GetTenantInfoByNamespace(api string, ns string) (TenantInfoList, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/tenants", t.tenantServiceURL), nil)
	if err != nil {
		return TenantInfoList{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.authToken))

	q := req.URL.Query()
	q.Add("master_url", util.EnsureSuffix(api, "/"))
	q.Add("namespace", ns)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return TenantInfoList{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return TenantInfoList{}, err
	}

	tenantInfo := TenantInfoList{}
	err = json.Unmarshal(body, &tenantInfo)
	if err != nil {
		return TenantInfoList{}, err
	}

	return tenantInfo, nil
}
