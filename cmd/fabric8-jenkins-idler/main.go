package main

import (
	"flag"
	"os"

	"context"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	openShiftClient "github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/token"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/version"
	log "github.com/sirupsen/logrus"
)

var mainLogger = log.WithFields(log.Fields{"component": "main"})

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
	mainLogger.Infof("Idler version: %s", version.GetVersion())

	// Init configuration
	config := createAndValidateConfiguration()
	mainLogger.Infof("Idler configuration: %s", config.String())

	// Get OSIO service account token from Auth
	osioToken := osioToken(config)

	// Get the view over the clusters
	clusterView := clusterView(osioToken, config)
	mainLogger.Infof("Cluster view: %s", clusterView.String())

	// Create Toggle (Unleash) Service
	featuresService := createFeatureToggle(config)

	// Create Tenant Service
	tenantService := tenant.NewTenantService(config.GetTenantURL(), osioToken)

	idler := NewIdler(featuresService, tenantService, clusterView, config)
	idler.Run()
}

func createAndValidateConfiguration() configuration.Configuration {
	var configFilePath string
	var printConfig bool
	flag.StringVar(&configFilePath, "config", "", "Path to the config file to read")
	flag.BoolVar(&printConfig, "printConfig", false, "Prints the config (including merged environment variables) and exits")
	flag.Parse()

	// Override default -config switch with environment variable only if -config switch was
	// not explicitly given via the command line.
	configSwitchIsSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			configSwitchIsSet = true
		}
	})
	if !configSwitchIsSet {
		if envConfigPath, ok := os.LookupEnv("F8_CONFIG_FILE_PATH"); ok {
			configFilePath = envConfigPath
		}
	}

	config, err := configuration.New(configFilePath)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"config_file_path": configFilePath,
			"err":              err,
		}, "failed to setup the configuration")
	}

	if printConfig {
		os.Exit(0)
	}

	multiError := config.Verify()
	if !multiError.Empty() {
		for _, err := range multiError.Errors {
			log.Error(err)
		}
		os.Exit(1)
	}
	return config
}

func createFeatureToggle(config configuration.Configuration) toggles.Features {
	var err error
	var features toggles.Features
	if len(config.GetFixedUuids()) > 0 {
		mainLogger.Infof("Using fixed UUID list for toggle feature: %s", config.GetFixedUuids())
		features, err = toggles.NewFixedUUIDToggle(config.GetFixedUuids())
	} else {
		features, err = toggles.NewUnleashToggle(config.GetToggleURL())
	}
	if err != nil {
		// Fatal with exit program
		mainLogger.WithField("err", err).Fatal("Unable to create feature toggles")
	}
	return features
}

func osioToken(config configuration.Configuration) string {
	osioToken, err := token.GetServiceAccountToken(config)
	if err != nil {
		// Fatal with exit program
		mainLogger.WithField("err", err).Fatal("Unable to retrieve service account token")
	}
	return osioToken
}

func clusterView(osioToken string, config configuration.Configuration) cluster.View {
	resolveToken := token.NewResolve(config.GetAuthURL())
	clusterService, err := cluster.NewService(
		config.GetAuthURL(),
		osioToken,
		resolveToken,
		token.NewPGPDecrypter(config.GetAuthTokenKey()),
		openShiftClient.NewOpenShift(),
	)
	if err != nil {
		// Fatal with exit program
		mainLogger.WithField("err", err).Fatal("Unable to create cluster service")
	}
	clusterView, err := clusterService.GetClusterView(context.Background())
	if err != nil {
		// Fatal with exit program
		mainLogger.WithField("err", err).Fatal("Unable to resolve cluster view")
	}

	return clusterView
}
