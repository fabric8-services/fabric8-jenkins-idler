package token

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/auth"
	authclient "github.com/fabric8-services/fabric8-jenkins-idler/internal/auth/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/pkg/errors"
)

// ServiceAccountTokenService the interface for the Service Account service
type ServiceAccountTokenService interface {
	GetOAuthToken(ctx context.Context) (*string, error)
}

// ServiceAccountTokenServiceConfig the config for the Service Account service
type ServiceAccountTokenServiceConfig interface {
	GetAuthURL() string
	GetServiceAccountID() string
	GetServiceAccountSecret() string
	GetAuthGrantType() string
}

// NewServiceAccountTokenService initializes a new ServiceAccountTokenService
func NewServiceAccountTokenService(config ServiceAccountTokenServiceConfig, options ...configuration.HTTPClientOption) ServiceAccountTokenService {
	return &serviceAccountTokenService{config: config}
}

// GetServiceAccountToken returns the OSIO service account token based on the passed configuration. If an error
// occurs the empty string together with the error are returned.
func GetServiceAccountToken(config configuration.Configuration) (string, error) {
	// fetch service account token for tenant service
	saTokenService := NewServiceAccountTokenService(config)
	saToken, err := saTokenService.GetOAuthToken(context.Background())
	if err != nil {
		return "", err
	}

	if saToken == nil {
		return "", fmt.Errorf("retrieved empty service account token")
	}

	return *saToken, nil
}

type serviceAccountTokenService struct {
	config        ServiceAccountTokenServiceConfig
	clientOptions []configuration.HTTPClientOption
}

func (s *serviceAccountTokenService) GetOAuthToken(ctx context.Context) (*string, error) {
	c, err := auth.NewClient(s.config.GetAuthURL(), "", s.clientOptions...) // no need to specify a token in this request
	if err != nil {
		return nil, errors.Wrapf(err, "error while initializing the auth client")
	}

	path := authclient.ExchangeTokenPath()
	payload := &authclient.TokenExchange{
		ClientID: s.config.GetServiceAccountID(),
		ClientSecret: func() *string {
			sec := s.config.GetServiceAccountSecret()
			return &sec
		}(),
		GrantType: s.config.GetAuthGrantType(),
	}
	contentType := "application/x-www-form-urlencoded"

	res, err := c.ExchangeToken(ctx, path, payload, contentType)
	if err != nil {
		return nil, errors.Wrapf(err, "error while doing the request")
	}
	defer func() {
		ioutil.ReadAll(res.Body)
		res.Body.Close()
	}()

	validationError := auth.ValidateResponse(c, res)
	if validationError != nil {
		return nil, errors.Wrapf(validationError, "error from server %q", s.config.GetAuthURL())
	}
	token, err := c.DecodeOauthToken(res)
	if err != nil {
		return nil, errors.Wrapf(err, "error from server %q", s.config.GetAuthURL())
	}

	if token.AccessToken == nil || *token.AccessToken == "" {
		return nil, fmt.Errorf("received empty token from server %q", s.config.GetAuthURL())
	}

	return token.AccessToken, nil
}
