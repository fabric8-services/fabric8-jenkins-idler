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
	"k8s.io/api/core/v1"
)

var logger = log.WithFields(log.Fields{"component": "openshift-client"})

// OpenShiftClient defines a stateless openShift client used to control namespace services in the specified cluster as well as
// monitoring a given cluster for events.
type OpenShiftClient interface {
	Idle(apiURL string, bearerToken string, namespace string, service string) error
	UnIdle(apiURL string, bearerToken string, namespace string, service string) error
	State(apiURL string, bearerToken string, namespace string, service string) (model.PodState, error)
	WhoAmI(apiURL string, bearerToken string) (string, error)
	WatchBuilds(apiURL string, bearerToken string, buildType string, callback func(model.Object) error) error
	WatchDeploymentConfigs(apiURL string, bearerToken string, namespaceSuffix string, callback func(model.DCObject) error) error
	Reset(apiURL string, bearerToken string, namespace string) error
}

type user struct {
	Metadata struct {
		Name string
	}
}

// openShift is a hand-rolled implementation of the OpenShiftClient using manually built-up HTTP requets.
type openShift struct {
	client *http.Client
}

// NewOpenShift creates new openShift client with new HTTP client.
func NewOpenShift() OpenShiftClient {
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(10) * time.Second,
	}

	return NewOpenShiftWithClient(c)
}

// NewOpenShiftWithClient create new openShift client with given HTTP client.
func NewOpenShiftWithClient(client *http.Client) OpenShiftClient {
	return &openShift{
		client: client,
	}
}

// Idle scales down the jenkins pod in the given openShift namespace.
func (o openShift) Idle(apiURL string, bearerToken string, namespace string, service string) (err error) {
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

	req, err := o.reqAPI(apiURL, bearerToken, "PATCH", namespace, fmt.Sprintf("endpoints/%s", service), br)
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

	req, err = o.reqOAPI(apiURL, bearerToken, "PATCH", namespace, fmt.Sprintf("deploymentconfigs/%s", service), br)
	if err != nil {
		return
	}
	b, err = o.patch(req)
	if err != nil {
		return
	}
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

// Reset deletes a pod and start a new one
func (o *openShift) Reset(apiURL string, bearerToken string, namespace string) error {
	logger.Infof("resetting pods in " + namespace)

	req, err := o.reqAPI(apiURL, bearerToken, "GET", namespace, "pods", nil)
	if err != nil {
		return err
	}
	resp, err := o.do(req)
	if err != nil {
		return err
	}

	defer bodyClose(resp)

	podList := &v1.PodList{}
	err = json.NewDecoder(resp.Body).Decode(podList)
	if err != nil {
		return err
	}

	for _, element := range podList.Items {

		podName := element.GetName()
		if strings.Contains(podName, "deploy") {
			continue
		}

		log.Infof("Resetting the pod %q", podName)
		req, err := o.reqAPI(apiURL, bearerToken, "DELETE", namespace, "pods/"+podName, nil)
		if err != nil {
			return err
		}

		resp, err = o.do(req)
		if err != nil {
			return err
		}
		defer bodyClose(resp)
	}
	return nil
}

// UnIdle scales up the jenkins pod in the given openShift namespace.
func (o *openShift) UnIdle(apiURL string, bearerToken string, namespace string, service string) (err error) {
	logger.Info("Un-idling ", service, " in ", namespace)
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
	req, err := o.reqOAPI(apiURL, bearerToken, "PUT", namespace, fmt.Sprintf("deploymentconfigs/%s/scale", service), br)
	if err != nil {
		return
	}
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

	logger.Infof("Scaled service %v to %v", service, ns.Spec.Replicas)
	return
}

// State returns `PodIdled` if a service in OpenShift namespace is idled,
// `PodStarting` if it is in the process of scaling up, `PodRunning`
// if it is fully up.
func (o *openShift) State(apiURL string, bearerToken string, namespace string, service string) (model.PodState, error) {
	req, err := o.reqOAPI(apiURL, bearerToken, "GET", namespace, "deploymentconfigs/"+service, nil)
	if err != nil {
		return model.PodStateUnknown, err
	}
	resp, err := o.do(req)
	if err != nil {
		return model.PodStateUnknown, err
	}

	defer bodyClose(resp)

	dc := &model.DeploymentConfig{}
	err = json.NewDecoder(resp.Body).Decode(dc)
	if err != nil {
		return model.PodStateUnknown, err
	}

	if dc.Status.Replicas == 0 {
		return model.PodIdled, nil
	}
	if dc.Status.ReadyReplicas == 0 {
		return model.PodStarting, nil
	}
	return model.PodRunning, nil
}

// GetScheme converts bool representing whether a route
// has TLS enabled to a web protocol string.
func (o openShift) getScheme(tls bool) string {
	scheme := "https"
	if !tls {
		scheme = "http"
	}

	return scheme
}

// WatchBuilds consumes stream of build events from openShift and calls callback to process them.
func (o openShift) WatchBuilds(apiURL string, bearerToken string, buildType string, callback func(model.Object) error) error {
	// Use a HTTP client with disabled timeout.
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(0) * time.Second,
	}
	for {
		req, err := o.reqOAPIWatch(apiURL, bearerToken, "GET", "", "builds", nil)
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
				// openShift sometimes ends the stream, break to create new request.
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

// WatchDeploymentConfigs consumes stream of DeploymentConfig events from openShift and calls callback to process them.
func (o openShift) WatchDeploymentConfigs(apiURL string, bearerToken string, namespaceSuffix string, callback func(model.DCObject) error) error {
	// Use a HTTP client with disabled timeout.
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(0) * time.Second,
	}
	for {
		req, err := o.reqOAPIWatch(apiURL, bearerToken, "GET", "", "deploymentconfigs", nil)
		if err != nil {
			logger.Fatal(err)
		}
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
			if !strings.HasSuffix(o.Object.Metadata.Namespace, namespaceSuffix) {
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

func (o openShift) WhoAmI(apiURL string, bearerToken string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apis/user.openshift.io/v1/users/~", strings.TrimSuffix(apiURL, "/")), nil)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve the username from the `whoami` API endpoint: %s", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve the username from the `whoami` API endpoint: %s", err)
	}
	defer resp.Body.Close()

	user := user{}
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return "", err
	}

	return user.Metadata.Name, nil
}

// GetBuilds loads builds for a given namespace from openShift.
func (o openShift) getBuilds(apiURL string, bearerToken string, namespace string) (bl model.BuildList, err error) {
	req, err := o.reqOAPI(apiURL, bearerToken, "GET", namespace, "builds", nil)
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

// req constructs a HTTP request for openShift API.
func (o *openShift) req(apiURL string, bearerToken string, method string, oapi bool, namespace string, command string, body io.Reader, watch bool) (req *http.Request, err error) {
	api := "api"
	if oapi {
		api = "oapi"
	}

	apiURL = strings.TrimSuffix(apiURL, "/")
	url := fmt.Sprintf("%s/%s/v1", apiURL, api)
	if len(namespace) > 0 {
		url = fmt.Sprintf("%s/%s/%s", url, "namespaces", namespace)
	}

	url = fmt.Sprintf("%s/%s", url, command)

	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	req.Header.Add("Authorization", "Bearer "+bearerToken)
	if watch {
		v := req.URL.Query()
		v.Add("watch", "true")
		req.URL.RawQuery = v.Encode()
	}

	return
}

// reqOAPI is a helper to construct a request for openShift API.
func (o *openShift) reqOAPI(apiURL string, bearerToken string, method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(apiURL, bearerToken, method, true, namespace, command, body, false)
}

// reqAPI is a help to construct a request for Kubernetes API.
func (o *openShift) reqAPI(apiURL string, bearerToken string, method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(apiURL, bearerToken, method, false, namespace, command, body, false)
}

// reqOAPIWatch is a helper to construct a request for openShift API using watch.
func (o *openShift) reqOAPIWatch(apiURL string, bearerToken string, method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(apiURL, bearerToken, method, true, namespace, command, body, true)
}

// reqAPIWatch is a helper to construct a request for Kubernetes API using watch.
func (o *openShift) reqAPIWatch(apiURL string, bearerToken string, method string, namespace string, command string, body io.Reader) (*http.Request, error) {
	return o.req(apiURL, bearerToken, method, false, namespace, command, body, true)
}

// do uses client.Do function to perform request and return response.
func (o *openShift) do(req *http.Request) (resp *http.Response, err error) {
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
func (o *openShift) patch(req *http.Request) (b []byte, err error) {
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

func bodyClose(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}
