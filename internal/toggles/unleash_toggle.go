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
