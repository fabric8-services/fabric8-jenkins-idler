package main

import (
	"os"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/version"
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
	log.Infof("Idler version: %s", version.GetVersion())

	//Init configuration
	config, err := configuration.NewConfiguration()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Idler configuration: %s", config.String())

	multiError := config.Verify()
	if !multiError.Empty() {
		for _, error := range multiError.Errors {
			log.Error(error)
		}
		os.Exit(1)
	}

	//Create Toggle (Unleash) Service client
	features, err := toggles.NewUnleashToggle(config.GetToggleURL())
	if err != nil {
		log.Fatal(err)
	}

	idler := NewIdler(config, features)
	idler.Run()
}
