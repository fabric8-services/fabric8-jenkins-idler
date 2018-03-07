package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{"component": "openshift-client"})

// OpenShiftClient is a client that talks to OpenShift API.
type OpenShiftClient interface {
	Idle(namespace string, service string) error
	UnIdle(namespace string, service string) error
	IsIdle(namespace string, service string) (int, error)
	GetRoute(n string, s string) (r string, tls bool, err error)
	GetAPIURL() string
	WatchBuilds(namespace string, buildType string, callback func(model.Object) error) error
	WatchDeploymentConfigs(namespace string, nsSuffix string, callback func(model.DCObject) error) error
}

// OpenShift is a client that talks to OpenShift API.
// It implements of OpenShiftClient.
type OpenShift struct {
	token  string
	apiURL string
	client *http.Client
}

// NewOpenShift creates new OpenShift client with new HTTP client.
func NewOpenShift(apiURL string, token string) OpenShiftClient {
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(10) * time.Second,
	}

	return NewOpenShiftWithClient(c, apiURL, token)
}

// NewOpenShiftWithClient create new OpenShift client with given HTTP client.
func NewOpenShiftWithClient(client *http.Client, apiURL string, token string) OpenShiftClient {
	if !strings.HasPrefix(apiURL, "http") {
		apiURL = fmt.Sprintf("https://%s", strings.TrimRight(apiURL, "/"))
	}
	return &OpenShift{
		apiURL: apiURL,
		token:  token,
		client: client,
	}
}

// Idle scales down the jenkins pod in the given OpenShift namespace.
func (o OpenShift) Idle(namespace string, service string) (err error) {
	logger.Info("Idling " + service + " in " + namespace)

	idleAt, err := time.Now().UTC().MarshalText()
	if err != nil {
		return
	}

	//Update annotations
	e := model.Endpoint{
		Metadata: model.Metadata{
			Annotations: model.Annotations{
				IdledAt:       string(idleAt),
				UnidleTargets: fmt.Sprintf("[{\"kind\":\"DeploymentConfig\",\"name\":\"%s\",\"replicas\":1}]", service),
			},
		},
	}
	body, err := json.Marshal(e)
	if err != nil {
		return
	}
	br := ioutil.NopCloser(bytes.NewReader(body))

	req, err := o.reqAPI("PATCH", namespace, fmt.Sprintf("endpoints/%s", service), br)
	if err != nil {
		return
	}
	b, err := o.patch(req)
	if err != nil {
		return
	}

	ne := &model.Endpoint{}
	err = json.Unmarshal(b, ne)
	if err != nil {
		return
	}

	// Check if returned object got updated.
	if e.Metadata.Annotations.IdledAt != string(idleAt) {
		return errors.New("could not update endpoint with idle time")
	}

	// Update DeploymentConfig - scale down.
	dc := model.DeploymentConfig{
		Metadata: model.Metadata{
			Annotations: model.Annotations{
				IdledAt:   string(idleAt),
				PrevScale: "1",
			},
		},
		Spec: model.Spec{
			Replicas: 0,
		},
	}
	body, err = json.Marshal(dc)
	if err != nil {
		return
	}
	br = ioutil.NopCloser(bytes.NewReader(body))

	req, err = o.reqOAPI("PATCH", namespace, fmt.Sprintf("deploymentconfigs/%s", service), br)
	if err != nil {
		return
	}
	b, err = o.patch(req)
	ndc := &model.DeploymentConfig{}
	err = json.Unmarshal(b, ndc)
	if err != nil {
		return
	}

	// Check successful scale-down.
	if ndc.Spec.Replicas != 0 {
		return errors.New("could not update DeploymentConfig with replica count")
	}

	return
}

// UnIdle scales up the jenkins pod in the given OpenShift namespace.
func (o *OpenShift) UnIdle(namespace string, service string) (err error) {
	logger.Info("Unidling ", service, " in ", namespace)
	// Scale up
	s := model.Scale{
		Kind:       "Scale",
		APIVersion: "extensions/v1beta1",
		Metadata: model.Metadata{
			Name:      service,
			Namespace: namespace,
		},
	}
	s.Spec.Replicas = 1
	body, err := json.Marshal(s)
	if err != nil {
		return
	}
	br := ioutil.NopCloser(bytes.NewReader(body))
	req, err := o.reqOAPI("PUT", namespace, fmt.Sprintf("deploymentconfigs/%s/scale", service), br)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.token))
	resp, err := o.client.Do(req)
	if err != nil {
		return
	}

	defer bodyClose(resp)

	ns := &model.Scale{}
	err = json.NewDecoder(resp.Body).Decode(ns)
	if err != nil {
		return
	}

	// Check new replica count.
	if ns.Spec.Replicas != s.Spec.Replicas {
		return errors.New("could not scale the service")
	}

	return
}

// IsIdle returns `JenkinsIdled` if a service in OpenShit namespace is idled,
// `JenkinsStarting` if it is in the process of scaling up, `JenkinsRunning`
// if it is fully up.
func (o *OpenShift) IsIdle(namespace string, service string) (int, error) {
	req, err := o.reqOAPI("GET", namespace, "deploymentconfigs/"+service, nil)
	if err != nil {
		return -1, err
	}
	resp, err := o.do(req)
	if err != nil {
		return -1, err
	}

	defer bodyClose(resp)

	dc := model.DeploymentConfig{}
	err = json.NewDecoder(resp.Body).Decode(&dc)
	if err != nil {
		return -1, err
	}

	if dc.Status.Replicas == 0 {
		return model.JenkinsIdled, nil
	}
	if dc.Status.ReadyReplicas == 0 {
		return model.JenkinsStarting, nil
	}
	return model.JenkinsRunning, nil
}

// GetRoute collects object for a given namespace and route name and returns
// url to reach it and if the route has enabled TLS.
func (o *OpenShift) GetRoute(n string, s string) (r string, tls bool, err error) {
	req, err := o.reqOAPI("GET", n, fmt.Sprintf("routes/%s", s), nil)
	if err != nil {
		return
	}
	resp, err := o.do(req)
	if err != nil {
		return
	}

	type route struct {
		Spec struct {
			Host string
			TLS  struct {
				Termination string
			} `json:"tls"`
		}
	}

	defer bodyClose(resp)

	rt := route{}
	err = json.NewDecoder(resp.Body).Decode(&rt)
	if err != nil {
		return
	}

	r = rt.Spec.Host
	tls = len(rt.Spec.TLS.Termination) > 0
	return
}

// GetScheme converts bool representing whether a route
// has TLS enabled to a web protocol string.
func (o OpenShift) getScheme(tls bool) string {
	scheme := "https"
	if !tls {
		scheme = "http"
	}

	return scheme
}

// GetProjects returns a list of projects which a user has access to.
func (o OpenShift) getProjects() (projects []string, err error) {
	req, err := o.reqOAPI("GET", "", "projects", nil)
	if err != nil {
		return
	}
	resp, err := o.do(req)
	if err != nil {
		return
	}

	defer bodyClose(resp)

	ps := model.Projects{}
	err = json.NewDecoder(resp.Body).Decode(&ps)
	if err != nil {
		return
	}

	for _, p := range ps.Items {
		if strings.HasSuffix(p.Metadata.Name, "-jenkins") {
			projects = append(projects, p.Metadata.Name[:len(p.Metadata.Name)-8])
		}
	}

	return
}

// WatchBuilds consumes stream of build events from OpenShift and calls callback to process them.
func (o OpenShift) WatchBuilds(namespace string, buildType string, callback func(model.Object) error) error {
	// Use a HTTP client with disabled timeout.
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(0) * time.Second,
	}
	for {
		req, err := o.reqOAPIWatch("GET", namespace, "builds", nil)
		if err != nil {
			logger.Fatal(err)
		}

		resp, err := c.Do(req)
		if err != nil {
			logger.Errorf("Request failed: %s", err)
			continue
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				// OpenShift sometimes ends the stream, break to create new request.
				if err.Error() == "EOF" || err.Error() == "unexpected EOF" {
					logger.Info("Got error ", err, " but continuing..")
					break
				}
			}

			o := model.Object{}

			err = json.Unmarshal(line, &o)
			if err != nil {
				// This happens with oc CLI tool as well from time to time, take care of it and create new request.
				if strings.HasPrefix(string(line), "This request caused apisever to panic") {
					logger.WithField("error", string(line)).Warning("Communication with server failed")
					break
				}
				logger.Errorf("Failed to Unmarshal: %s", err)
				break
			}

			// Verify a build has a type we care about.
			if o.Object.Spec.Strategy.Type != buildType {
				logger.WithField("namespace", o.Object.Metadata.Namespace).Debugf("Skipping build %s (type: %s)", o.Object.Metadata.Name, o.Object.Spec.Strategy.Type)
				continue
			}
			logger.WithFields(log.Fields{"namespace": o.Object.Metadata.Namespace, "data": o}).Debug("Handling Build change event")
			err = callback(o)
			if err != nil {
				logger.Errorf("Error from callback: %s", err)
				continue
			}
		}
		logger.Debug("Fell out of loop for Build")
	}
}

// WatchDeploymentConfigs consumes stream of DeploymentConfig events from OpenShift and calls callback to process them.
func (o OpenShift) WatchDeploymentConfigs(namespace string, nsSuffix string, callback func(model.DCObject) error) error {
	// Use a HTTP client with disabled timeout.
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(0) * time.Second,
	}
	for {
		req, err := o.reqOAPIWatch("GET", namespace, "deploymentconfigs", nil)
		if err != nil {
			logger.Fatal(err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", o.token))
		v := req.URL.Query()
		v.Add("labelSelector", "app=jenkins")
		req.URL.RawQuery = v.Encode()
		resp, err := c.Do(req)
		if err != nil {
			logger.Errorf("Request failed: %s", err)
			continue
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err.Error() == "EOF" || err.Error() == "unexpected EOF" {
					logger.Info("Got error ", err, " but continuing..")
					break
				}
				fmt.Printf("It's broken %+v", err)
			}

			o := model.DCObject{}

			err = json.Unmarshal(line, &o)
			if err != nil {
				if strings.HasPrefix(string(line), "This request caused apisever to panic") {
					logger.WithField("error", string(line)).Warning("Communication with server failed")
					break
				}
				logger.Errorf("Failed to Unmarshal: %s", err)
				break
			}

			// Filter for a given suffix.
			if !strings.HasSuffix(o.Object.Metadata.Namespace, nsSuffix) {
				logger.WithField("namespace", o.Object.Metadata.Namespace).Debug("Skipping DC change event")
				continue
			}

			logger.WithFields(log.Fields{"namespace": o.Object.Metadata.Namespace, "data": o}).Debug("Handling DC change event")
			err = callback(o)
			if err != nil {
				logger.Errorf("Error from DC callback: %s", err)
				continue
			}
		}
		logger.Debug("Fell out of loop for watching DC")
	}
}

// GetBuilds loads builds for a given namespace from OpenShift.
func (o OpenShift) getBuilds(namespace string) (bl model.BuildList, err error) {
	req, err := o.reqOAPI("GET", namespace, "builds", nil)
	if err != nil {
		return
	}

	resp, err := o.do(req)
	if err != nil {
		return
	}

	defer bodyClose(resp)
	err = json.NewDecoder(resp.Body).Decode(&bl)
	return
}

// req construcs a HTTP request for OpenShift/Kubernetes API.
func (o *OpenShift) req(method string, oapi bool, namespace string, command string, body io.Reader, watch bool) (req *http.Request, err error) {
	api := "api"
	if oapi {
		api = "oapi"
	}

	url := fmt.Sprintf("%s/%s/v1", o.apiURL, api)
	if len(namespace) > 0 {
		url = fmt.Sprintf("%s/%s/%s", url, "namespaces", namespace)
	}

	url = fmt.Sprintf("%s/%s", url, command)

	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	req.Header.Add("Authorization", "Bearer "+o.token)
	if watch {
		v := req.URL.Query()
		v.Add("watch", "true")
		req.URL.RawQuery = v.Encode()
	}

	return
}

// reqOAPI is a helper to construct a request for OpenShift API.
func (o *OpenShift) reqOAPI(method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(method, true, namespace, command, body, false)
}

// reqAPI is a help to construct a request for Kubernetes API.
func (o *OpenShift) reqAPI(method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(method, false, namespace, command, body, false)
}

// reqOAPIWatch is a helper to construct a request for OpenShift API using watch.
func (o *OpenShift) reqOAPIWatch(method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(method, true, namespace, command, body, true)
}

// reqAPIWatch is a helper to construct a request for Kubernetes API using watch.
func (o *OpenShift) reqAPIWatch(method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(method, false, namespace, command, body, true)
}

// do uses client.Do function to perform request and return response.
func (o *OpenShift) do(req *http.Request) (resp *http.Response, err error) {
	resp, err = o.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("got status %s (%d) from %s", resp.Status, resp.StatusCode, req.URL)
	}

	return
}

// patch is a helper to perform a PATCH request.
func (o *OpenShift) patch(req *http.Request) (b []byte, err error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/strategic-merge-patch+json")

	resp, err := o.do(req)
	if err != nil {
		return
	}

	defer bodyClose(resp)
	b, err = ioutil.ReadAll(resp.Body)
	return
}

// GetAPIURL returns API Url for OpenShift cluster.
func (o *OpenShift) GetAPIURL() string {
	return o.apiURL
}

func bodyClose(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}
