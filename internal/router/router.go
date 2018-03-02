package router

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

var routerLogger = log.WithFields(log.Fields{"component": "router"})

const (
	defaultHTTPServerPort = 8080
	shutdownTimeout       = 5
)

// Router implements an HTTP server, exposing the REST API of the Idler
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

// Start starts the HTTP router
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

// Shutdown shuts down the idler router
func (r *Router) Shutdown() {
	routerLogger.Info("Idler router shutting down.")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout*time.Second)
	if err := r.srv.Shutdown(ctx); err != nil {
		routerLogger.Error(err) // failure/timeout shutting down the server gracefully
	}
	cancel()
}

// CreateAPIRouter creates idler api router
func CreateAPIRouter(api api.IdlerAPI) *httprouter.Router {
	router := httprouter.New()

	router.GET("/iapi/idler/builds/:namespace", api.User)
	router.GET("/iapi/idler/builds/:namespace/", api.User)

	router.GET("/iapi/idler/idle/:namespace", api.Idle)
	router.GET("/iapi/idler/idle/:namespace/", api.Idle)

	router.GET("/iapi/idler/unidle/:namespace", api.UnIdle)
	router.GET("/iapi/idler/unidle/:namespace/", api.UnIdle)

	router.GET("/iapi/idler/isidle/:namespace", api.IsIdle)
	router.GET("/iapi/idler/isidle/:namespace/", api.IsIdle)

	return router
}
