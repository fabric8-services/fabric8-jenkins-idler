package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_string_representation_of_cluster(t *testing.T) {
	cluster := Cluster{}

	actualString := cluster.String()
	expectedString := `[APIURL:  ConsoleURL:  MetricsURL:  LoggingURL:  AppDNS:  Type: ]`
	assert.Equal(t, actualString, expectedString)
}
