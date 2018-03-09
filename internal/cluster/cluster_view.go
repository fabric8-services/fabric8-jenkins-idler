package cluster

import "fmt"

// View provides a view over the current cluster topology.
type View interface {
	GetClusters() []Cluster
	GetDNSView() []DNSView
	GetToken(openShiftAPIURL string) (string, bool)
	String() string
}

// DNSView is a view of the cluster topology which only includes the OpenShift API URL and the application DNS for this
// cluster.
type DNSView struct {
	APIURL string
	AppDNS string
}

// NewView returns a new instance of View.
func NewView(clusters []Cluster) View {
	return &clusterView{
		clusters: clusters,
	}
}

type clusterView struct {
	clusters []Cluster
}

func (c clusterView) GetClusters() []Cluster {
	return c.clusters
}

func (c clusterView) GetDNSView() []DNSView {
	var dnsClusters []DNSView

	for _, cluster := range c.clusters {
		dnsCluster := DNSView{
			APIURL: cluster.APIURL,
			AppDNS: cluster.AppDNS,
		}
		dnsClusters = append(dnsClusters, dnsCluster)
	}
	return dnsClusters
}

func (c clusterView) GetToken(openShiftAPIURL string) (string, bool) {
	for _, cluster := range c.clusters {
		if cluster.APIURL == openShiftAPIURL {
			return cluster.Token, true
		}
	}
	return "", false
}

func (c clusterView) String() string {
	return fmt.Sprintf("%v", c.clusters)
}
