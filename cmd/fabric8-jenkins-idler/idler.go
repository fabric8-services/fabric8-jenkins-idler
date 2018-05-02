package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"

	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/cluster"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/router"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/tenant"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

const (
	profilerPort = 6060
)

var idlerLogger = log.WithFields(log.Fields{"component": "idler"})

// Idler is responsible to create and control the various concurrent processes needed to implement the Jenkins idling
// feature. An Idler instance creates two goroutines for watching all builds respectively deployment config changes of
// the whole cluster. To do this it needs an access openShift access token which allows the Idler to do so (see Data.GetOpenShiftToken).
// A third go routine is used to serve a HTTP REST API.
type Idler struct {
	featureService toggles.Features
	tenantService  tenant.Service
	clusterView    cluster.View
	config         configuration.Configuration
}

// NewIdler creates a new instance of Idler. The configuration as well as feature toggle handler needs to be passed.
func NewIdler(features toggles.Features, tenantService tenant.Service, clusterView cluster.View, config configuration.Configuration) *Idler {
	return &Idler{
		featureService: features,
		tenantService:  tenantService,
		clusterView:    clusterView,
		config:         config,
	}
}

// Run starts the various goroutines of the Idler. To cleanly shutdown the SIGTERM signal should be send to the process.
func (idler *Idler) Run() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	idler.startWorkers(ctx, &wg, cancel, idler.config.GetDebugMode())

	setupSignalChannel(cancel)
	wg.Wait()
	idlerLogger.Info("Idler successfully shut down.")
}

func (idler *Idler) startWorkers(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, addProfiler bool) {
	idlerLogger.Info("Starting all Idler workers")

	// Create synchronized map for UserIdler instances
	userIdlers := openshift.NewUserIdlerMap()

	// Start the controllers to monitor the OpenShift clusters
	idler.startOpenShiftControllers(ctx, wg, cancel, userIdlers)

	// Start API router
	go func() {
		// Create and start a Router instance to serve the REST API
		idlerAPI := api.NewIdlerAPI(userIdlers, idler.clusterView, idler.tenantService)
		apirouter := router.CreateAPIRouter(idlerAPI)
		router := router.NewRouter(apirouter)
		router.AddMetrics(apirouter)
		router.Start(ctx, wg, cancel)
	}()

	if addProfiler {
		go func() {
			idlerLogger.Infof("Starting profiler on port %d", profilerPort)
			router := router.NewRouterWithPort(httprouter.New(), profilerPort)
			router.Start(ctx, wg, cancel)
		}()
	}
}

func (idler *Idler) startOpenShiftControllers(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, userIdlers *openshift.UserIdlerMap) {
	openShiftClient := client.NewOpenShift()

	for _, openShiftCluster := range idler.clusterView.GetClusters() {
		// Create Controller
		controller := openshift.NewController(
			ctx,
			openShiftCluster.APIURL,
			openShiftCluster.Token,
			userIdlers,
			idler.tenantService,
			idler.featureService,
			idler.config,
			wg,
			cancel,
		)

		wg.Add(1)
		go func(cluster cluster.Cluster) {
			defer wg.Done()
			go func() {
				idlerLogger.Info("Starting to watch openShift deployment configuration changes.")
				if err := openShiftClient.WatchDeploymentConfigs(cluster.APIURL, cluster.Token, "-jenkins", controller.HandleDeploymentConfig); err != nil {
					cancel()
					return
				}
			}()

			for {
				select {
				case <-ctx.Done():
					idlerLogger.Infof("Stopping to watch openShift deployment configuration changes.")
					cancel()
					return
				}
			}
		}(openShiftCluster)

		wg.Add(1)
		go func(cluster cluster.Cluster) {
			defer wg.Done()
			go func() {
				idlerLogger.Info("Starting to watch openShift build configuration changes.")
				if err := openShiftClient.WatchBuilds(cluster.APIURL, cluster.Token, "JenkinsPipeline", controller.HandleBuild); err != nil {
					cancel()
					return
				}
			}()

			for {
				select {
				case <-ctx.Done():
					idlerLogger.Infof("Stopping to watch openShift build configuration changes.")
					cancel()
					return
				}
			}
		}(openShiftCluster)
	}
}

// setupSignalChannel registers a listener for Unix signals for a ordered shutdown
func setupSignalChannel(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		idlerLogger.Info("Received SIGTERM signal. Initiating shutdown.")
		cancel()
	}()
}
