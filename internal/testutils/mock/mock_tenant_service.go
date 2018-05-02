package mock

import "github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"

// TenantService provides information about tenants running on Openshift cluster.
// This is the mock interface
type TenantService struct{}

// GetTenantInfoByNamespace Mocks get info
func (t *TenantService) GetTenantInfoByNamespace(apiURL string, ns string) (tenant.InfoList, error) {
	return tenant.InfoList{}, nil
}

// HasReachedMaxCapacity returns false all  the time
func (t *TenantService) HasReachedMaxCapacity(apiURL, ns string) (bool, error) {
	return false, nil
}
