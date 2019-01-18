package cluster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	authClient "github.com/fabric8-services/fabric8-jenkins-idler/internal/auth/client"
	openShiftClient "github.com/fabric8-services/fabric8-jenkins-idler/internal/openshift/client"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/token"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_clusterService_GetClusterView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := NewMockauthService(ctrl)

	ocClient := openShiftClient.NewMockOpenShiftClient(ctrl)

	ctx := context.Background()

	type fields struct {
		resolveToken token.Resolve
	}
	tests := []struct {
		name     string
		fields   fields
		preqFunc func()
		want     View
		wantErr  bool
	}{
		{
			name: "Cluster Service OSO No Err",
			fields: fields{
				resolveToken: func(ctx context.Context, target, token string, forcePull bool, decode token.Decode) (username, accessToken string, err error) {
					return "test", "test", nil
				},
			},
			preqFunc: func() {
				res := &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte("test"))),
				}

				client.EXPECT().ShowClusters(ctx,
					authClient.ShowClustersPath()).Return(res, nil)
				clusters := authClient.ClusterList{
					Data: []*authClient.ClusterData{
						{
							APIURL: "http://apiurl",
							Type:   "OSO",
						},
					},
				}

				client.EXPECT().DecodeClusterList(res).Return(&clusters, nil)

				ocClient.EXPECT().WhoAmI("http://apiurl", "test").Return("", nil)
			},
			want: &clusterView{clusters: []Cluster{
				{

					APIURL: "http://apiurl",
					User:   "test",
					Token:  "test",
					Type:   "OSO",
				},
			}},
			wantErr: false,
		},
		{
			name: "Cluster Service OSO Show Cluster http Status UnAuth",
			fields: fields{
				resolveToken: func(ctx context.Context, target, token string, forcePull bool, decode token.Decode) (username, accessToken string, err error) {
					return "test", "test", nil
				},
			},
			preqFunc: func() {
				client.EXPECT().ShowClusters(ctx,
					authClient.ShowClustersPath()).Return(nil,
					errors.New("Unauthorized"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Cluster Service OSO DecodeJSONAPIErrors Err",
			fields: fields{
				resolveToken: func(ctx context.Context, target, token string, forcePull bool, decode token.Decode) (username, accessToken string, err error) {
					return "test", "test", nil
				},
			},
			preqFunc: func() {
				res := &http.Response{
					StatusCode: http.StatusBadRequest,
					Body: ioutil.NopCloser(
						bytes.NewReader([]byte("Bad Response"))),
				}

				client.EXPECT().ShowClusters(ctx,
					authClient.ShowClustersPath()).Return(res, nil)
				client.EXPECT().DecodeJSONAPIErrors(res).Return(nil,
					errors.New("Bad Res"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Cluster Service OSO NotFound",
			fields: fields{
				resolveToken: func(ctx context.Context, target, token string, forcePull bool, decode token.Decode) (username, accessToken string, err error) {
					return "test", "test", nil
				},
			},
			preqFunc: func() {
				res := &http.Response{
					StatusCode: http.StatusNotFound,
					Body: ioutil.NopCloser(
						bytes.NewReader([]byte("Bad Response"))),
				}

				client.EXPECT().ShowClusters(ctx,
					authClient.ShowClustersPath()).Return(res, nil)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Cluster Service OSO DecodeClusterList Err",
			fields: fields{
				resolveToken: func(ctx context.Context, target, token string, forcePull bool, decode token.Decode) (username, accessToken string, err error) {
					return "test", "test", nil
				},
			},
			preqFunc: func() {
				res := &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte("test"))),
				}

				client.EXPECT().ShowClusters(ctx,
					authClient.ShowClustersPath()).Return(res, nil)
				client.EXPECT().DecodeClusterList(res).Return(nil, errors.New("Decode Err ClusterList"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Cluster Service OSO resolveToken Err",
			fields: fields{
				resolveToken: func(ctx context.Context, target,
					token string, forcePull bool,
					decode token.Decode) (username,
					accessToken string, err error) {
					return "", "", errors.New("UnAuth")
				},
			},
			preqFunc: func() {
				res := &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte("test"))),
				}

				client.EXPECT().ShowClusters(ctx,
					authClient.ShowClustersPath()).Return(res, nil)
				client.EXPECT().DecodeClusterList(res).Return(nil, errors.New("Decode Err ClusterList"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Cluster Service OSO OC Client WhoAmI Err",
			fields: fields{
				resolveToken: func(ctx context.Context, target, token string, forcePull bool, decode token.Decode) (username, accessToken string, err error) {
					return "test", "test", nil
				},
			},
			preqFunc: func() {
				res := &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader([]byte("test"))),
				}

				client.EXPECT().ShowClusters(ctx,
					authClient.ShowClustersPath()).Return(res, nil)
				clusters := authClient.ClusterList{
					Data: []*authClient.ClusterData{
						{
							APIURL: "http://apiurl",
							Type:   "OSO",
						},
					},
				}

				client.EXPECT().DecodeClusterList(res).Return(&clusters, nil)

				ocClient.EXPECT().WhoAmI("http://apiurl", "test").Return("",
					errors.New("UnAuth"))
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &clusterService{
				authClient:   client,
				ocClient:     ocClient,
				resolveToken: tt.fields.resolveToken,
			}
			tt.preqFunc()
			got, err := s.GetClusterView(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("clusterService.GetClusterView() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, got, tt.want,
				fmt.Sprintf("clusterService.GetClusterView() = %v, want %v",
					got, tt.want))
		})
	}
}
