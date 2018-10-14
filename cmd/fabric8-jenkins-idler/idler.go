package main

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"

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
// the whole cluster. To do this it needs an access openshift access token which allows the Idler to do so (see Data.GetOpenShiftToken).
// A third go routine is used to serve a HTTP REST API.
type Idler struct {
	featureService toggles.Features
	tenantService  tenant.Service
	clusterView    cluster.View
	config         configuration.Configuration
}

// struct used to pass in cancelable task
type task struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	t := &task{ctx, cancel, &wg}
	setupSignalChannel(t)

	idler.startWorkers(t, idler.config.GetDebugMode())
	wg.Wait()
	idlerLogger.Info("Idler successfully shut down.")
}

func (idler *Idler) startWorkers(t *task, addProfiler bool) {
	idlerLogger.Info("Starting all Idler workers")

	// Create synchronized map for UserIdler instances
	userIdlers := openshift.NewUserIdlerMap()

	// Start the controllers to monitor the OpenShift clusters
	idler.watchOpenshiftEvents(t, userIdlers)

	// Start API router
	go func() {
		// Create and start a Router instance to serve the REST API
		idlerAPI := api.NewIdlerAPI(userIdlers, idler.clusterView, idler.tenantService)
		apirouter := router.CreateAPIRouter(idlerAPI)
		router := router.NewRouter(apirouter)
		router.AddMetrics(apirouter)
		router.Start(t.ctx, t.wg, t.cancel)
	}()

	if addProfiler {
		go func() {
			idlerLogger.Infof("Starting profiler on port %d", profilerPort)
			router := router.NewRouterWithPort(httprouter.New(), profilerPort)
			router.Start(t.ctx, t.wg, t.cancel)
		}()
	}
}

func (idler *Idler) watchOpenshiftEvents(t *task, userIdlers *openshift.UserIdlerMap) {
	oc := client.NewOpenShift()

	for _, c := range idler.clusterView.GetClusters() {
		// Create Controller
		ctrl := openshift.NewController(
			t.ctx,
			c.APIURL,
			c.Token,
			userIdlers,
			idler.tenantService,
			idler.featureService,
			idler.config,
			t.wg,
			t.cancel,
		)

		t.wg.Add(1)
		go idler.watchBC(t, oc, c, ctrl.HandleBuild)
	}
}

type bcHandler func(model.Object) error

func (idler *Idler) watchBC(t *task, oc client.OpenShiftClient, c cluster.Cluster, handler bcHandler) {
	defer t.wg.Done()
	go func() {
		idlerLogger.Info("Starting to watch openshift build configuration changes.")
		err := oc.WatchBuilds(c.APIURL, c.Token, "JenkinsPipeline", handler)
		if err != nil {
			t.cancel()
		}
	}()

	<-t.ctx.Done()
	idlerLogger.Infof("Stopping to watch openshift build configuration changes.")
	t.cancel()
}

// setupSignalChannel registers a listener for Unix signals for a ordered shutdown
func setupSignalChannel(t *task) {
	t.wg.Add(1)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	go func() {
		defer func() {
			t.cancel()
			t.wg.Done()
		}()

		select {
		case <-sigChan:
			idlerLogger.Info("Received SIGTERM signal. Initiating shutdown.")
		case <-t.ctx.Done():
			idlerLogger.Info("Context got cancelled. Initiating shutdown.")
		}
	}()
}
