package toggles

import (
	"os"
	"time"

	unleash "github.com/Unleash/unleash-client-go"
	ucontext "github.com/Unleash/unleash-client-go/context"
	log "github.com/sirupsen/logrus"
)

var ready = false

// Init toggle client lib
func Init(serviceName, hostURL string) {
	unleash.Initialize(
		unleash.WithAppName(serviceName),
		unleash.WithInstanceId(os.Getenv("HOSTNAME")),
		unleash.WithUrl(hostURL),
		unleash.WithMetricsInterval(1*time.Minute),
		unleash.WithRefreshInterval(10*time.Second),
		unleash.WithListener(&listener{}),
	)

	ready = false
}

// WithContext creates a Token based contex
func WithContext(sub string) unleash.FeatureOption {
	uctx := ucontext.Context{
		UserId: sub,
	}

	return unleash.WithContext(uctx)
}

// IsEnabled wraps unleash for a simpler API
func IsEnabled(sub string, feature string, fallback bool) bool {
	if !ready {
		return fallback
	}
	return unleash.IsEnabled(feature, WithContext(sub), unleash.WithFallback(fallback))
}

type listener struct{}

// OnError prints out errors.
func (l listener) OnError(err error) {
	log.Error(nil, map[string]interface{}{
		"err": err.Error(),
	}, "toggles error")
}

// OnWarning prints out warning.
func (l listener) OnWarning(warning error) {
	log.Warn(nil, map[string]interface{}{
		"err": warning.Error(),
	}, "toggles warning")
}

// OnReady prints to the console when the repository is ready.
func (l listener) OnReady() {
	ready = true
	log.Info(nil, map[string]interface{}{}, "toggles ready")
}

// OnCount prints to the console when the feature is queried.
func (l listener) OnCount(name string, enabled bool) {
	log.Info(nil, map[string]interface{}{
		"name":    name,
		"enabled": enabled,
	}, "toggles count")
}

// OnSent prints to the console when the server has uploaded metrics.
func (l listener) OnSent(payload unleash.MetricsData) {
	log.Info(nil, map[string]interface{}{
		"payload": payload,
	}, "toggles sent")
}

// OnRegistered prints to the console when the client has registered.
func (l listener) OnRegistered(payload unleash.ClientData) {
	log.Info(nil, map[string]interface{}{
		"payload": payload,
	}, "toggles registered")
}

func IsReady() bool {
	return ready
}
