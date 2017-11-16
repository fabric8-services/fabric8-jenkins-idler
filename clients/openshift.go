package clients

import (
	"bytes"
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

type OpenShift struct {
	token string
	apiURL string
	client http.Client
}

func NewOpenShift(apiURL string, token string) OpenShift {
	if !strings.HasPrefix(apiURL, "http") {
		apiURL = fmt.Sprintf("https://%s", strings.TrimRight(apiURL, "/"))
	}
	return OpenShift{
		apiURL: apiURL,
		token: token,
		client: http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
			},
			Timeout: time.Duration(5) * time.Second,
		},
	}
}

func (o OpenShift) Idle(namespace string, service string) (err error) {
	log.Info("Idling "+service+" in "+namespace)

	idleAt, err := time.Now().UTC().MarshalText()
	if err != nil {
		return
	}
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

	if e.Metadata.Annotations.IdledAt != string(idleAt) {
		return errors.New("Could not update endpoint with idle time.")
	}

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

	if ndc.Spec.Replicas != 0 {
		return errors.New("Could not update DeploymentConfig with replica count.")
	}

	return
}

func (o *OpenShift) UnIdle(namespace string, service string) (err error) {
	log.Info("Unidling ", service, " in ", namespace)
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

	if ns.Spec.Replicas != s.Spec.Replicas {
		return errors.New("Could not scale the service")
	}

	return
}

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
		return JenkinsStates["Idle"], nil
	}

	if dc.Status.ReadyReplicas == 0 {
		return JenkinsStates["Starting"], nil
	}

	return JenkinsStates["Running"], nil
}

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

func (o *OpenShift) req(method string, oapi bool, namespace string, command string, body io.Reader) (req *http.Request, err error) {
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

	return
}

func (o *OpenShift) reqOAPI(method string, namespace string, command string, body io.Reader) (req *http.Request, err error) {
	return o.req(method, true, namespace, command, body)
}

func (o *OpenShift) reqAPI(method string, namespace string, command string, body io.Reader) (req *http.Request, err error) {
	return o.req(method, false, namespace, command, body)
}


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

func bodyClose(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}