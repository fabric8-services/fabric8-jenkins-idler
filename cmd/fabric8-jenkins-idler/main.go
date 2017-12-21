package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	_, ok := os.LookupEnv("JC_LOCAL_DEV_ENV")
	// TODO - remove this and make it a proper test/testprogram
	if ok {
		testutils.Run()
		return
	}

	idler := NewIdler()
	idler.Run()
}
