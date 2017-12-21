package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
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

	//Init configuration
	config, err := configuration.NewData()
	if err != nil {
		log.Fatal(err)
	}
	config.Verify()

	//Create Toggle (Unleash) Service client
	features, err := toggles.NewUnleashToggle(config.GetToggleURL())
	if err != nil {
		log.Fatal(err)
	}

	idler := NewIdler(config, features)
	idler.Run()
}
