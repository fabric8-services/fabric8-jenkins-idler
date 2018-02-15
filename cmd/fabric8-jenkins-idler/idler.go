package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"

	proxyClient "github.com/fabric8-services/fabric8-jenkins-proxy/clients"

	"context"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/router"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	profilerPort = 6060
)

var mainLogger = log.WithFields(log.Fields{"component": "main"})

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
	openShift := client.NewOpenShift(idler.config.GetOpenShiftURL(), idler.config.GetOpenShiftToken())
	tenantClient := proxyClient.NewTenant(idler.config.GetTenantURL(), idler.config.GetAuthToken())

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Idler controller
	controller := openshift.NewOpenShiftController(
		openShift,
		&tenantClient,
		idler.features,
		idler.config,
		&wg,
		ctx,
		cancel,
	)

	startWorkers(&wg, ctx, cancel, openShift, controller, idler.config.GetDebugMode())

	setupSignalChannel(cancel)
	wg.Wait()
	mainLogger.Info("Idler successfully shutdown.")
}

func startWorkers(wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc, openShift client.OpenShiftClient, controller openshift.Controller, addProfiler bool) {
	mainLogger.Info("Starting  all workers")

	// Start API router
	go func() {
		// Create and start a Router instance to serve the REST API
		idlerApi := api.NewIdlerAPI(openShift, controller)
		router := router.NewRouter(router.CreateAPIRouter(idlerApi))

		router.Start(wg, ctx, cancel)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		go func() {
			mainLogger.Info("Starting to watch OpenShift deployment configuration changes.")
			if err := openShift.WatchDeploymentConfigs("", "-jenkins", controller.HandleDeploymentConfig); err != nil {
				cancel()
				return
			}
		}()

		for {
			select {
			case <-ctx.Done():
				mainLogger.Infof("Stopping to watch OpenShift deployment configuration changes.")
				cancel()
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		go func() {
			mainLogger.Info("Starting to watch OpenShift build configuration changes.")
			if err := openShift.WatchBuilds("", "JenkinsPipeline", controller.HandleBuild); err != nil {
				cancel()
				return
			}
		}()

		for {
			select {
			case <-ctx.Done():
				mainLogger.Infof("Stopping to watch OpenShift build configuration changes.")
				cancel()
				return
			}
		}
	}()

	if addProfiler {
		go func() {
			mainLogger.Infof("Starting profiler on port %d", profilerPort)
			router := router.NewRouterWithPort(httprouter.New(), profilerPort)
			router.Start(wg, ctx, cancel)
		}()
	}
}

// setupSignalChannel registers a listener for Unix signals for a ordered shutdown
func setupSignalChannel(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		mainLogger.Info("Received SIGTERM signal. Initiating shutdown.")
		cancel()
	}()
}
