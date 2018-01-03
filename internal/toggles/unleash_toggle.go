package toggles

import (
	"os"
	"time"

	"errors"
	"fmt"

	unleash "github.com/Unleash/unleash-client-go"
	"github.com/Unleash/unleash-client-go/context"
	log "github.com/sirupsen/logrus"
)

const (
	appName         = "jenkins-idler"
	toggleFeature   = "jenkins.idler"
	maxWaitForReady = 10
)

type unleashToggle struct {
	Features
	unleashClient *unleash.Client
}

func NewUnleashToggle(hostURL string) (Features, error) {
	unleashClient, err := unleash.NewClient(unleash.WithAppName(appName),
		unleash.WithListener(&listener{}),
		unleash.WithInstanceId(os.Getenv("HOSTNAME")),
		unleash.WithUrl(hostURL),
		unleash.WithMetricsInterval(1*time.Minute),
		unleash.WithRefreshInterval(10*time.Second))

	if err != nil {
		log.Error("Unable to initialize Unleash client.", err)
		return nil, err
	}

	readyChan := unleashClient.Ready()
	select {
	case <-readyChan:
		log.Info("Unleash client initalized and ready.")
	case <-time.After(time.Second * maxWaitForReady):
		return nil, errors.New(fmt.Sprintf("Unleash client initalization timed out after %d seconds.", maxWaitForReady))
	}

	return &unleashToggle{unleashClient: unleashClient}, nil
}

func (t *unleashToggle) IsIdlerEnabled(uid string) (bool, error) {
	enabled := t.unleashClient.IsEnabled(toggleFeature, t.withContext(uid), unleash.WithFallback(false))
	return enabled, nil
}

// withContext creates a context based toggle with the user id as key.
func (t *unleashToggle) withContext(uid string) unleash.FeatureOption {
	ctx := context.Context{
		UserId: uid,
	}

	return unleash.WithContext(ctx)
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
	log.Info(nil, map[string]interface{}{}, "toggles ready")
}

// OnCount prints to the console when the feature is queried.
func (l listener) OnCount(name string, enabled bool) {
	log.Debug(nil, map[string]interface{}{
		"name":    name,
		"enabled": enabled,
	}, "toggles count")
}

// OnSent prints to the console when the server has uploaded metrics.
func (l listener) OnSent(payload unleash.MetricsData) {
	log.Debug(nil, map[string]interface{}{
		"payload": payload,
	}, "toggles sent")
}

// OnRegistered prints to the console when the client has registered.
func (l listener) OnRegistered(payload unleash.ClientData) {
	log.Info(nil, map[string]interface{}{
		"payload": payload,
	}, "toggles registered")
}
