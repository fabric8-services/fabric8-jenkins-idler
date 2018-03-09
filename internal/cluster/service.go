package cluster

import (
	"context"
	"io/ioutil"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/auth"
	authClient "github.com/fabric8-services/fabric8-jenkins-idler/internal/auth/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	openShiftClient "github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/token"
	goaclient "github.com/goadesign/goa/client"
	"github.com/pkg/errors"
	"strings"
)

// Service the interface for the cluster service
type Service interface {
	GetClusterView(context.Context) (View, error)
}

// NewService creates a Resolver that rely on the Auth service to retrieve tokens
func NewService(authURL string, serviceToken string, resolveToken token.Resolve, decode token.Decode, options ...configuration.HTTPClientOption) Service {
	return &clusterService{authURL: authURL, serviceToken: serviceToken, resolveToken: resolveToken, decode: decode, clientOptions: options}
}

type clusterService struct {
	authURL       string
	clientOptions []configuration.HTTPClientOption
	serviceToken  string
	resolveToken  token.Resolve
	decode        token.Decode
}

func cleanURL(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

func (s *clusterService) GetClusterView(ctx context.Context) (View, error) {
	client, err := auth.NewClient(s.authURL, s.serviceToken, s.clientOptions...)
	if err != nil {
		return nil, err
	}
	client.SetJWTSigner(
		&goaclient.JWTSigner{
			TokenSource: &goaclient.StaticTokenSource{
				StaticToken: &goaclient.StaticToken{
					Value: s.serviceToken,
					Type:  "Bearer"}}})

	res, err := client.ShowClusters(ctx, authClient.ShowClustersPath())
	if err != nil {
		return nil, errors.Wrapf(err, "error while doing the request")
	}
	defer func() {
		ioutil.ReadAll(res.Body)
		res.Body.Close()
	}()

	validationError := auth.ValidateResponse(client, res)
	if validationError != nil {
		return nil, errors.Wrapf(validationError, "error from server %q", s.authURL)
	}

	clusters, err := client.DecodeClusterList(res)
	if err != nil {
		return nil, errors.Wrapf(err, "error from server %q", s.authURL)
	}

	var clusterList []Cluster
	for _, cluster := range clusters.Data {
		// resolve/obtain the cluster token
		clusterUser, clusterToken, err := s.resolveToken(ctx, cluster.APIURL, s.serviceToken, false, s.decode) // can't use "forcePull=true" to validate the `tenant service account` token since it's encrypted on auth
		if err != nil {
			return nil, errors.Wrapf(err, "unable to resolve token for cluster %v", cluster.APIURL)
		}

		// verify the token
		openShiftClient := openShiftClient.NewOpenShift()
		_, err = openShiftClient.WhoAmI(cluster.APIURL, clusterToken)
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
		})
	}
	return NewView(clusterList), nil
}
