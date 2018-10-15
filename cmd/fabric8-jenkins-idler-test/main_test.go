package main

import (
	"os"
	"testing"
)

func XTestSingleIdler(t *testing.T) {

	os.Setenv("JC_DEBUG_MODE", "true")
	os.Setenv("JC_IDLE_AFTER", "5")
	os.Setenv("JC_CHECK_INTERVAL", "5")
	os.Setenv("JC_MAX_RETRIES", "5")
	os.Setenv("JC_MAX_RETRIES_QUIET_INTERVAL", "5")

	idler := NewTestIdler(baseName, identityID, clusterURL, token)
	idler.Run()
}
