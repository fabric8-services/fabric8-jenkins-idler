package cluster

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_string_representation_of_cluster(t *testing.T) {
	cluster := Cluster{}

	actualString := cluster.String()
	expectedString := `[APIURL:  ConsoleURL:  MetricsURL:  LoggingURL:  AppDNS: ]`
	assert.Equal(t, actualString, expectedString)
}
