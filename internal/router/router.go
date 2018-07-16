package router

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var routerLogger = log.WithFields(log.Fields{"component": "router"})

const (
	defaultHTTPServerPort = 8080
	shutdownTimeout       = 5
)

// Router implements an HTTP server, exposing the REST API of the Idler.
type Router struct {
	port int
	srv  *http.Server
}

// NewRouter creates a new HTTP router for the Idler on the default port.
func NewRouter(router *httprouter.Router) *Router {
	return NewRouterWithPort(router, defaultHTTPServerPort)
}

// NewRouterWithPort creates a new HTTP router for the Idler on the specified port.
func NewRouterWithPort(router *httprouter.Router, port int) *Router {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	return &Router{port: port, srv: srv}
}

// AddMetrics add metrics handler to serve promotheus metrics
func (r *Router) AddMetrics(router *httprouter.Router) {
	router.Handler("GET", "/metrics", prometheus.Handler())
}

// Start starts the HTTP router.
func (r *Router) Start(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		go func() {
			routerLogger.Infof("Starting API router on port %d.", r.port)
			if err := r.srv.ListenAndServe(); err != nil {
				cancel()
				return
			}
		}()

		for {
			select {
			case <-ctx.Done():
				routerLogger.Infof("Shutting down API router on port %d.", r.port)
				ctx, cancel := context.WithTimeout(ctx, shutdownTimeout*time.Second)
				r.srv.Shutdown(ctx)
				cancel()
				return
			}
		}
	}()
}

// Shutdown shuts down the idler router.
func (r *Router) Shutdown() {
	routerLogger.Info("Idler router shutting down.")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout*time.Second)
	if err := r.srv.Shutdown(ctx); err != nil {
		routerLogger.Error(err) // failure/timeout shutting down the server gracefully.
	}
	cancel()
}

// CreateAPIRouter a pointer to the http router for the idler APIs.
func CreateAPIRouter(api api.IdlerAPI) *httprouter.Router {
	router := httprouter.New()

	router.GET("/api/idler/info/:namespace", api.Info)
	router.GET("/api/idler/info/:namespace/", api.Info)

	router.GET("/api/idler/idle/:namespace", api.Idle)
	router.GET("/api/idler/idle/:namespace/", api.Idle)

	router.GET("/api/idler/unidle/:namespace", api.UnIdle)
	router.GET("/api/idler/unidle/:namespace/", api.UnIdle)

	router.GET("/api/idler/isidle/:namespace", api.IsIdle)
	router.GET("/api/idler/isidle/:namespace/", api.IsIdle)

	router.GET("/api/idler/status/:namespace", api.Status)
	router.GET("/api/idler/status/:namespace/", api.Status)

	router.GET("/api/idler/cluster", api.ClusterDNSView)
	router.GET("/api/idler/cluster/", api.ClusterDNSView)

	router.POST("/api/idler/reset/:namespace", api.Reset)
	router.POST("/api/idler/reset/:namespace/", api.Reset)

	return router
}
