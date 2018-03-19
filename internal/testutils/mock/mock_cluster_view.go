package mock

import "github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"

// ClusterView provides a view over the current cluster topology.
// This is the mock interface
type ClusterView struct{}

// GetClusters get cluster view
func (m *ClusterView) GetClusters() (r []cluster.Cluster) {
	return
}

// GetDNSView get dns view
func (m *ClusterView) GetDNSView() (r []cluster.DNSView) {
	return
}

// GetToken get token of cluster
func (m *ClusterView) GetToken(openShiftAPIURL string) (string, bool) {
	return "", true
}

func (m *ClusterView) String() string {
	return ""
}
