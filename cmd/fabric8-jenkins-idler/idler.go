package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"

	proxyClient "github.com/fabric8-services/fabric8-jenkins-proxy/clients"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/router"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

// Idler is responsible to create and control the various concurrent processes needed to implement the Jenkins idling
// feature. An Idler instance creates two goroutines for watching all builds respectively deployment config changes of
// the whole cluster. To do this it needs an access OpenShift access token which allows the Idler to do so (see Data.GetOpenShiftToken).
// A third go routine is used to serve a HTTP REST API.
type Idler struct {
	features toggles.Features
	config   configuration.Configuration
}

// NewIdler creates a new instance of Idler. The configuration as well as feature toggle handler needs to be passed.
func NewIdler(config configuration.Configuration, features toggles.Features) *Idler {
	return &Idler{
		config:   config,
		features: features,
	}
}

// Run starts the various goroutines of the Idler. To cleanly shutdown the SIGTERM signal should be send to the process.
func (idler *Idler) Run() {
	openShift := openshift.NewOpenShift(idler.config.GetOpenShiftURL(), idler.config.GetOpenShiftToken())
	tenantClient := proxyClient.NewTenant(idler.config.GetTenantURL(), idler.config.GetAuthToken())

	// Create Idler controller
	controller := openshift.NewOpenShiftController(
		openShift,
		&tenantClient,
		idler.features,
		idler.config,
	)

	// Start the controller
	controller.Run()

	// Create a done channel to signal goroutines a shutdown
	done := make(chan interface{})

	// Create and start a Router instance to serve the REST API
	idlerApi := api.NewIdlerAPI(openShift, controller)
	router := router.NewRouter(idlerApi)
	terminated := router.Start(done)

	setupSignalChannel(done)

	<-terminated
	log.Info("Idler shutdown complete.")
}

// setupSignalChannel registers a listener for Unix signals for a ordered shutdown
func setupSignalChannel(done chan interface{}) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM)

	go func() {
		<-sigchan
		log.Info("Received SIGTERM signal. Initiating shutdown.")
		done <- true
	}()
}
