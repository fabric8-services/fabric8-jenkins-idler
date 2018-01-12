package router

import (
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/api"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"net/http"
	"time"
)

var routerLogger = log.WithFields(log.Fields{"component": "router"})

const (
	defaultHttpServerPort = 8080
	shutdownTimeout       = 5
)

// Router implements an HTTP server, exposing the REST API of the Idler
type Router struct {
	port int
	srv  *http.Server
}

// NewRouter creates a new HTTP router for the Idler on the default port.
func NewRouter(api api.IdlerAPI) *Router {
	return NewRouterWithPort(api, defaultHttpServerPort)
}

// NewRouter creates a new HTTP router for the Idler on the specified port.
func NewRouterWithPort(api api.IdlerAPI, port int) *Router {
	router := createRouter(api)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	return &Router{port: port, srv: srv}
}

// Start starts the HTTP router
func (r *Router) Start(done <-chan interface{}) <-chan interface{} {
	terminated := make(chan interface{})
	go func() {
		go func() {
			defer routerLogger.Infof("Idler router stopped.")
			defer close(terminated)
			routerLogger.Infof("Starting Idler router on port %d.", r.port)
			err := r.srv.ListenAndServe()
			routerLogger.Errorf("ListenAndServe() error in Idler: %s", err)
		}()

		// waiting on the done channel
		<-done
		r.Shutdown()
	}()

	return terminated
}

func (r *Router) Shutdown() {
	routerLogger.Info("Idler router shutting down.")
	ctx, _ := context.WithTimeout(context.Background(), shutdownTimeout*time.Second)
	if err := r.srv.Shutdown(ctx); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
}

func createRouter(api api.IdlerAPI) *httprouter.Router {
	router := httprouter.New()

	router.GET("/iapi/idler/builds", api.User)
	router.GET("/iapi/idler/builds/", api.User)

	router.GET("/iapi/idler/builds/:namespace", api.User)
	router.GET("/iapi/idler/builds/:namespace/", api.User)

	router.GET("/iapi/idler/idle/:namespace", api.Idle)
	router.GET("/iapi/idler/idle/:namespace/", api.Idle)

	router.GET("/iapi/idler/isidle/:namespace", api.IsIdle)
	router.GET("/iapi/idler/isidle/:namespace/", api.IsIdle)

	router.GET("/iapi/idler/route/:namespace", api.GetRoute)
	router.GET("/iapi/idler/route/:namespace/", api.GetRoute)

	return router
}
