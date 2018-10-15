package main

import (
	"io/ioutil"
	"syscall"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/mock"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

type mockFeatureToggle struct {
}

func (m *mockFeatureToggle) IsIdlerEnabled(uid string) (bool, error) {
	return true, nil
}

type mockClusterView struct {
	*mock.ClusterView
}

func (mc *mockClusterView) GetClusters() []cluster.Cluster {
	var clusters []cluster.Cluster
	// dummy cluster
	cluster := cluster.Cluster{
		APIURL: "http://127.0.0.1",
		Token:  "abc",
	}
	clusters = append(clusters, cluster)

	return clusters
}

func Test_graceful_shutdown(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	// register a global log hook to capture the log output
	hook := test.NewGlobal()

	config, _ := configuration.NewConfiguration()
	idler := NewIdler(&mockFeatureToggle{}, &mock.TenantService{}, &mockClusterView{}, config)

	go func() {
		// Send SIGTERM after two seconds
		time.Sleep(3 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	idler.Run()

	logMessages := extractLogMessages(hook.AllEntries())
	assert.Contains(t, logMessages, "Stopping to watch openshift build configuration changes.", "Idler shutdown completion should have been logged")
	assert.Contains(t, logMessages, "Idler successfully shut down.", "Idler shutdown completion should have been logged")
}

func extractLogMessages(entries []*log.Entry) []string {
	var messages []string
	for _, logEntry := range entries {
		messages = append(messages, logEntry.Message)
	}
	return messages
}
