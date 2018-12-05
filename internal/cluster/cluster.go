package cluster

import "fmt"

// Cluster describes a cluster in the system with its relevant information, in particular the various URLs.
type Cluster struct {
	APIURL     string
	ConsoleURL string
	MetricsURL string
	LoggingURL string
	AppDNS     string

	User  string
	Token string
	Type  string
}

func (c Cluster) String() string {
	var tmp []string

	tmp = append(tmp, fmt.Sprintf("APIURL: %s", c.APIURL))
	tmp = append(tmp, fmt.Sprintf("ConsoleURL: %s", c.ConsoleURL))
	tmp = append(tmp, fmt.Sprintf("MetricsURL: %s", c.MetricsURL))
	tmp = append(tmp, fmt.Sprintf("LoggingURL: %s", c.LoggingURL))
	tmp = append(tmp, fmt.Sprintf("AppDNS: %s", c.AppDNS))

	return fmt.Sprintf("%+v", tmp)
}
