package cluster

import (
	"context"
	"io/ioutil"

	"strings"

	"net/http"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/auth"
	authClient "github.com/fabric8-services/fabric8-jenkins-idler/internal/auth/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	openShiftClient "github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/token"
	goaclient "github.com/goadesign/goa/client"
	"github.com/pkg/errors"
)

const (
	osioType = "OSO"
)

// Service the interface for the cluster service
type Service interface {
	GetClusterView(context.Context) (View, error)
}

// NewService creates a Resolver that rely on the Auth service to retrieve tokens
func NewService(authURL, serviceToken string, resolveToken token.Resolve,
	decode token.Decode, ocClient openShiftClient.OpenShiftClient,
	options ...configuration.HTTPClientOption) (Service, error) {

	client, err := auth.NewClient(authURL, serviceToken, options...)
	if err != nil {
		return nil, err
	}
	client.SetJWTSigner(
		&goaclient.JWTSigner{
			TokenSource: &goaclient.StaticTokenSource{
				StaticToken: &goaclient.StaticToken{
					Value: serviceToken,
					Type:  "Bearer"}}})

	return &clusterService{authURL: authURL, serviceToken: serviceToken,
		resolveToken: resolveToken, decode: decode, ocClient: ocClient,
		clientOptions: options, authClient: client}, nil
}

type authService interface {
	auth.ValidateAuth
	ShowClusters(ctx context.Context, path string) (*http.Response, error)
	DecodeClusterList(resp *http.Response) (*authClient.ClusterList, error)
}

type clusterService struct {
	authURL       string
	clientOptions []configuration.HTTPClientOption
	serviceToken  string
	resolveToken  token.Resolve
	decode        token.Decode
	authClient    authService
	ocClient      openShiftClient.OpenShiftClient
}

func cleanURL(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

func (s *clusterService) GetClusterView(ctx context.Context) (View, error) {
	res, err := s.authClient.ShowClusters(ctx, authClient.ShowClustersPath())
	if err != nil {
		return nil, errors.Wrapf(err, "error while doing the request")
	}
	defer func() {
		ioutil.ReadAll(res.Body)
		res.Body.Close()
	}()

	validationError := auth.ValidateResponse(s.authClient, res)
	if validationError != nil {
		return nil, errors.Wrapf(validationError, "error from server %q", s.authURL)
	}

	clusters, err := s.authClient.DecodeClusterList(res)
	if err != nil {
		return nil, errors.Wrapf(err, "error from server %q", s.authURL)
	}

	var clusterList []Cluster
	for _, cluster := range clusters.Data {
		if cluster.Type != osioType {
			continue
		}
		// resolve/obtain the cluster token
		clusterUser, clusterToken, err := s.resolveToken(ctx, cluster.APIURL, s.serviceToken, false, s.decode) // can't use "forcePull=true" to validate the `tenant service account` token since it's encrypted on auth
		if err != nil {
			return nil, errors.Wrapf(err, "unable to resolve token for cluster %v", cluster.APIURL)
		}

		// verify the token
		_, err = s.ocClient.WhoAmI(cluster.APIURL, clusterToken)
		if err != nil {
			return nil, errors.Wrapf(err, "token retrieved for cluster %v is invalid", cluster.APIURL)
		}

		if err != nil {
			return nil, errors.Wrapf(err, "token retrieved for cluster %v is invalid", cluster.APIURL)
		}

		clusterList = append(clusterList, Cluster{
			APIURL:     cluster.APIURL,
			AppDNS:     cluster.AppDNS,
			ConsoleURL: cluster.ConsoleURL,
			MetricsURL: cluster.MetricsURL,
			LoggingURL: cluster.LoggingURL,
			User:       clusterUser,
			Token:      clusterToken,
			Type:       cluster.Type,
		})
	}
	return NewView(clusterList), nil
}
