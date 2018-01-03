package main

import (
	"os"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	level := log.InfoLevel
	switch levelStr, _ := os.LookupEnv("JC_LOG_LEVEL"); levelStr {
	case "info":
		level = log.InfoLevel
	case "debug":
		level = log.DebugLevel
	case "warning":
		level = log.WarnLevel
	case "error":
		level = log.ErrorLevel
	default:
		level = log.InfoLevel
	}
	log.SetLevel(level)
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
