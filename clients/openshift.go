package clients

import (
	"bytes"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"io"
	"io/ioutil"
	"time"
	"strings"

	log "github.com/sirupsen/logrus"
)

//OpenShift is a client for OpenShift API
type OpenShift struct {
	token string
	apiURL string
	client *http.Client
}

//NewOpenShift creates new OpenShift client with new http client
func NewOpenShift(apiURL string, token string) OpenShift {
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(10) * time.Second,
	}

	return NewOpenShiftWithClient(c, apiURL, token)
}

//NewOpenShiftWithClient create new OpenShift client with given http client
func NewOpenShiftWithClient(client *http.Client, apiURL string, token string) OpenShift {
	if !strings.HasPrefix(apiURL, "http") {
		apiURL = fmt.Sprintf("https://%s", strings.TrimRight(apiURL, "/"))
	}
	return OpenShift{
		apiURL: apiURL,
		token: token,
		client: client,
	}
}

//Idle forces a service in OpenShift namespace to idle
func (o OpenShift) Idle(namespace string, service string) (err error) {
	log.Info("Idling "+service+" in "+namespace)

	idleAt, err := time.Now().UTC().MarshalText()
	if err != nil {
		return
	}

	//Update annotations
	e := Endpoint{
		Metadata: Metadata{
			Annotations: Annotations{
				IdledAt: string(idleAt),
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

	ne := &Endpoint{}
	err = json.Unmarshal(b, ne)
	if err != nil {
		return
	}

	//Check if returned object got updated
	if e.Metadata.Annotations.IdledAt != string(idleAt) {
		return errors.New("Could not update endpoint with idle time.")
	}

	//Update DeploymentConfig - scale down
	dc := DeploymentConfig{
		Metadata: Metadata{
			Annotations: Annotations{
				IdledAt: string(idleAt),
				PrevScale: "1",
			},
		},
		Spec: Spec{
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
	ndc := &DeploymentConfig{}
	err = json.Unmarshal(b, ndc)
	if err != nil {
		return
	}

	//Check successful scale-down
	if ndc.Spec.Replicas != 0 {
		return errors.New("Could not update DeploymentConfig with replica count.")
	}

	return
}

//Unidle forces a service in OpenShift namespace to start
func (o *OpenShift) UnIdle(namespace string, service string) (err error) {
	log.Info("Unidling ", service, " in ", namespace)
	//Scale up
	s := Scale{
		Kind: "Scale",
		ApiVersion: "extensions/v1beta1",
		Metadata: Metadata {
			Name: service,
			Namespace: namespace,
		},
	}
	s.Spec.Replicas = 1
	body, err := json.Marshal(s)
	if err != nil {
		return
	}
	br := ioutil.NopCloser(bytes.NewReader(body))
	req, err := o.reqOAPI("PUT", namespace, fmt.Sprintf("deploymentconfigs/%s/scale", service), br) //FIXME
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.token))
	resp, err := o.client.Do(req)
	if err != nil {
		return
	}

	defer bodyClose(resp)

	ns := &Scale{}
	err = json.NewDecoder(resp.Body).Decode(ns)
	if err != nil {
		return
	}

	//Check new replica count
	if ns.Spec.Replicas != s.Spec.Replicas {
		return errors.New("Could not scale the service")
	}

	return
}

//IsIdle returns `JenkinsIdled` if a service in OpenShit namespace is idled,
//`JenkinsStarting` if it is in the process of scaling up, `JenkinsRunning`
//if it is fully up
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

	dc := DeploymentConfig{}
	err = json.NewDecoder(resp.Body).Decode(&dc)
	if err != nil {
		return -1, err
	}

	if dc.Status.Replicas == 0 {
		return JenkinsIdled, nil
	}
	if dc.Status.ReadyReplicas == 0 {
		return JenkinsStarting, nil
	}
	return JenkinsRunning, nil
}

//GetRoute collects object for a given namespace and route name and returns
//url to reach it and if the route has enabled TLS
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
			TLS struct {
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

//GetScheme converts bool representing whether a route
//has TLS enabled to a web protocol string
func (o OpenShift) GetScheme(tls bool) string {
	scheme := "https"
	if !tls {
		scheme = "http"
	}

	return scheme
}

//GetProjects returns a list of projects which a user has access to
func (o OpenShift) GetProjects() (projects []string, err error) {
	req, err := o.reqOAPI("GET", "", "projects", nil)
	if err != nil {
		return
	}
	resp, err := o.do(req)
	if err != nil {
		return
	}

	defer bodyClose(resp)

	ps := Projects{}
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

//WatchBuilds consumes stream of build events from OpenShift and calls callback to process them
func (o OpenShift) WatchBuilds(namespace string, buildType string, callback func(Object) (bool, error)) (err error) {
	//Use a http client with disabled timeout
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(0) * time.Second,
	}
	for {
		req, err := o.reqOAPIWatch("GET", namespace, "builds", nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", o.token))

		resp, err := c.Do(req)
		if err != nil {
			log.Errorf("Request failed: %s", err)
			continue
		}

		reader := bufio.NewReader(resp.Body)
		for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					//OpenShift sometimes ends the stream, break to create new request
					if err.Error() == "EOF" || err.Error() == "unexpected EOF" {
						log.Info("Got error ", err, " but continuing..")
						break;
					}
				}

				o := Object{}

				err = json.Unmarshal(line, &o)
				if err!=nil {
					//This happens with oc CLI tool as well from time to time, take care of it and create new request
					if strings.HasPrefix(string(line), "This request caused apisever to panic") {
						log.WithField("error", string(line)).Warning("Communication with server failed")
						break;
					}
					log.Errorf("Failed to Unmarshal: %s", err)
					break
				}

				//Verify a build has a type we care about
				if o.Object.Spec.Strategy.Type != buildType {
					log.Infof("Skipping build %s (type: %s)", o.Object.Metadata.Name, o.Object.Spec.Strategy.Type)
					continue
				}
				log.Infof("Handling Build change for user %s", o.Object.Metadata.Namespace)
				ok, err := callback(o)
				if err != nil {
					log.Errorf("Error from callback: %s", err)
					continue
				}

				if ok {
					log.Debugf("Event summary: Build %s -> %s, %s/%s", o.Object.Metadata.Name, o.Object.Status.Phase, o.Object.Status.StartTimestamp, o.Object.Status.CompletionTimestamp) 
				}
		}
	}

	return
}

//WatchDeploymentConfigs consumes stream of DC events from OpenShift and calls callback to process them; FIXME - a lot of copy&paste from
//watch builds, refactor!
func (o OpenShift) WatchDeploymentConfigs(namespace string, nsSuffix string, callback func(DCObject) (bool, error)) (err error) {
	//Use a http client with disabled timeout
	c := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Duration(0) * time.Second,
	}
	for {
		req, err := o.reqOAPIWatch("GET", namespace, "deploymentconfigs", nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", o.token))
		v := req.URL.Query()
		v.Add("labelSelector", "app=jenkins")
		req.URL.RawQuery = v.Encode()
		resp, err := c.Do(req)
		if err != nil {
			log.Errorf("Request failed: %s", err)
			continue
		}
	
		reader := bufio.NewReader(resp.Body)
		for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					if err.Error() == "EOF" || err.Error() == "unexpected EOF" {	
						log.Info("Got error ", err, " but continuing..")
						break;
					} 
					fmt.Printf("It's broken %+v\n", err)
				}

				o := DCObject{}
			
				err = json.Unmarshal(line, &o)
				if err!=nil {
					if strings.HasPrefix(string(line), "This request caused apisever to panic") {
						log.WithField("error", string(line)).Warning("Communication with server failed")
						break;
					}
					log.Errorf("Failed to Unmarshal: %s", err)
					break
				}

				//Filter for a given suffix
				if !strings.HasSuffix(o.Object.Metadata.Namespace, nsSuffix) {
					log.Infof("Skipping DC %s", o.Object.Metadata.Namespace)
					continue
				}

				log.Infof("Handling DC change for user %s\n", o.Object.Metadata.Namespace)
				ok, err := callback(o)
				if err != nil {
					log.Errorf("Error from DC callback: %s", err)
					continue
				}

				if ok {
					//Get piece of status for debug info;FIXME - should this go away or at least be conditional?
					c, err := o.Object.Status.GetByType("Available")
					if err != nil {
						log.Error(err)
						continue
					}

					log.Debugf("Event summary: DeploymentConfig %s, %s/%s\n", o.Object.Metadata.Name, c.Status, c.LastUpdateTime) 
				}
		}
	}

	return
}

//GetBuilds loads builds for a given namespace from OpenShift
func (o OpenShift) GetBuilds(namespace string) (bl BuildList, err error) {
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

//req construcs a http request for OpenShift/Kubernetes API
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

//reqOAPI is a helper to construct a request for OpenShift API
func (o *OpenShift) reqOAPI(method string, namespace string, command string, body io.Reader) (req *http.Request, err error) {
	return o.req(method, true, namespace, command, body, false)
}

//reqAPI is a help to construct a request for Kubernetes API
func (o *OpenShift) reqAPI(method string, namespace string, command string, body io.Reader) (req *http.Request, err error) {
	return o.req(method, false, namespace, command, body, false)
}

//reqOAPIWatch is a helper to construct a request for OpenShift API using watch
func (o *OpenShift) reqOAPIWatch(method string, namespace string, command string, body io.Reader) (req *http.Request, err error) {
	return o.req(method, true, namespace, command, body, true)
}

//reqAPIWatch is a helper to construct a request for Kubernetes API using watch
func (o *OpenShift) reqAPIWatch(method string, namespace string, command string, body io.Reader) (req *http.Request, err error) {
	return o.req(method, false, namespace, command, body, true)
}

//do uses client.Do function to perform request and return response
func (o *OpenShift) do(req *http.Request) (resp *http.Response, err error) {
	resp, err = o.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("Got status %s (%d) from %s.", resp.Status, resp.StatusCode, req.URL))
	}

	return
}

//patch is a helper to perform a PATCH request
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

func (o *OpenShift) GetApiURL() string {
	return o.apiURL
}

func bodyClose(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}