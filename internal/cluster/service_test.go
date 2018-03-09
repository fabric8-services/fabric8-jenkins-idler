package cluster_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterCache(t *testing.T) {
	t.Run("cluster - end slash", func(t *testing.T) {
		// given
		target := "A"
		resolve := cluster.NewResolve([]*cluster.Cluster{
			{APIURL: "X"},
			{APIURL: target + "/"},
		})
		// when
		found, err := resolve(context.Background(), target)
		// then
		require.NoError(t, err)
		assert.Contains(t, found.APIURL, target)
	})

	t.Run("cluster - no end slash", func(t *testing.T) {
		// given
		target := "A"
		resolve := cluster.NewResolve([]*cluster.Cluster{
			{APIURL: "X"},
			{APIURL: target},
		})
		// when
		found, err := resolve(context.Background(), target+"/")
		// then
		require.NoError(t, err)
		assert.Contains(t, found.APIURL, target)
	})

	t.Run("both slash", func(t *testing.T) {
		// given
		target := "A"
		resolve := cluster.NewResolve([]*cluster.Cluster{
			{APIURL: "X"},
			{APIURL: target + "/"},
		})
		// when
		found, err := resolve(context.Background(), target+"/")
		// then
		require.NoError(t, err)
		assert.Contains(t, found.APIURL, target)
	})

	t.Run("no slash", func(t *testing.T) {
		// given
		target := "A"
		resolve := cluster.NewResolve([]*cluster.Cluster{
			{APIURL: "X"},
			{APIURL: target + "/"},
		})
		// when
		found, err := resolve(context.Background(), target+"/")
		// then
		require.NoError(t, err)
		assert.Contains(t, found.APIURL, target)
	})
}
