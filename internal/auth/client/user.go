// Code generated by goagen v1.3.0, DO NOT EDIT.
//
// API "auth": user Resource Client
//
// Command:
// $ goagen
// --design=github.com/fabric8-services/fabric8-auth/design
// --out=$(GOPATH)/src/github.com/fabric8-services/fabric8-auth
// --version=v1.3.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ListResourcesUserPath computes a request path to the listResources action of user.
func ListResourcesUserPath() string {

	return fmt.Sprintf("/api/user/resources")
}

// List resources of a given type with a role for the current user
func (c *Client) ListResourcesUser(ctx context.Context, path string, type_ string) (*http.Response, error) {
	req, err := c.NewListResourcesUserRequest(ctx, path, type_)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewListResourcesUserRequest create the request corresponding to the listResources action endpoint of the user resource.
func (c *Client) NewListResourcesUserRequest(ctx context.Context, path string, type_ string) (*http.Request, error) {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "http"
	}
	u := url.URL{Host: c.Host, Scheme: scheme, Path: path}
	values := u.Query()
	values.Set("type", type_)
	u.RawQuery = values.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.JWTSigner != nil {
		c.JWTSigner.Sign(req)
	}
	return req, nil
}

// ShowUserPath computes a request path to the show action of user.
func ShowUserPath() string {

	return fmt.Sprintf("/api/user")
}

// Get the authenticated user in JSON-API format
func (c *Client) ShowUser(ctx context.Context, path string, ifModifiedSince *string, ifNoneMatch *string) (*http.Response, error) {
	req, err := c.NewShowUserRequest(ctx, path, ifModifiedSince, ifNoneMatch)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewShowUserRequest create the request corresponding to the show action endpoint of the user resource.
func (c *Client) NewShowUserRequest(ctx context.Context, path string, ifModifiedSince *string, ifNoneMatch *string) (*http.Request, error) {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "http"
	}
	u := url.URL{Host: c.Host, Scheme: scheme, Path: path}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	header := req.Header
	if ifModifiedSince != nil {

		header.Set("If-Modified-Since", *ifModifiedSince)
	}
	if ifNoneMatch != nil {

		header.Set("If-None-Match", *ifNoneMatch)
	}
	if c.JWTSigner != nil {
		c.JWTSigner.Sign(req)
	}
	return req, nil
}
