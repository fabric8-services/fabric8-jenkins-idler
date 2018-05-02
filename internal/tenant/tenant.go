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
	HasReachedMaxCapacity(apiURL, ns string) (bool, error)
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

// returns true if the cluster the ns is on has reached maximum capacity
func (t tenantService) HasReachedMaxCapacity(apiURL, ns string) (bool, error) {

	ti, err := t.GetTenantInfoByNamespace(apiURL, ns)
	if err != nil {
		return true, err
	}

	if len(ti.Errors) != 0 {
		firstError := ti.Errors[0]
		err = fmt.Errorf("%s - %s", firstError.Code, firstError.Detail)
		return true, err
	}

	if len(ti.Data) == 0 {
		err = fmt.Errorf("Failed to fetch tenant information for %s", ns)
		return true, err
	}

	namespaces := ti.Data[0].Attributes.Namespaces
	index := indexOfNamespaceWithName(namespaces, ns)
	if index < 0 {
		return true, fmt.Errorf("Failed to find %s in tenant info", ns)
	}

	jenkins := namespaces[index]
	return jenkins.ClusterCapacityExhausted, nil
}

// returns the index of the namespace that equals 'name'
func indexOfNamespaceWithName(namespaces []Namespace, name string) int {
	for i, ns := range namespaces {
		if ns.Name == name {
			return i
		}
	}
	return -1
}
