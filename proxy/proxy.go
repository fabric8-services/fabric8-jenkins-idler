package proxy

import (
	"time"
	"crypto/tls"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	oc "github.com/fabric8-services/fabric8-jenkins-idler/openshiftcontroller"
	log "github.com/sirupsen/logrus"
)


type Proxy struct {
	RequestBuffer map[string]*[]BufferedReuqest
	OC *oc.OpenShiftController
	bufferLock *sync.Mutex
	newUrl string
	service string
	token string
}

type BufferedReuqest struct {
	Request *http.Request
	Body []byte
}

func NewProxy(oc *oc.OpenShiftController, token string) Proxy {
	rb := make(map[string]*[]BufferedReuqest)
	p := Proxy{
		RequestBuffer: rb,
		OC: oc,
		bufferLock: &sync.Mutex{},
		newUrl: "jenkins-%s-jenkins.d800.free-int.openshiftapps.com", //"content-repository-%s-jenkins.d800.free-int.openshiftapps.com", //
		service: "jenkins",
		token: token,
	}
	go func() {
		p.ProcessBuffer()
	}()
	return p
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Host: %s\nPath: %s\n", r.Host, r.URL.Path)
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
			log.Fatal(err)
	}

	isGH := false
	if ua, exist := r.Header["User-Agent"]; exist {
		isGH = strings.HasPrefix(ua[0], "GitHub-Hookshot")
	}

	if isGH {
		payload := loadHookPayload(body)
		name := p.GetUser(payload.Repository.FullName)
		namespace := fmt.Sprintf("%s-jenkins", name)
		r.Host = fmt.Sprintf(p.newUrl, name)

		if p.OC.IsIdle(namespace, p.service) {
			w.Header().Set("Server", "Webhook-Proxy")
			if !p.OC.UnIdle(namespace, p.service) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(""))
				return
			}
			p.bufferLock.Lock()

			if _, exist := p.RequestBuffer[name]; !exist {
				rbs := make([]BufferedReuqest, 0, 50)
				p.RequestBuffer[name] = &rbs
			}
			rb := p.RequestBuffer[name]
			*p.RequestBuffer[name] = append(*rb, BufferedReuqest{Request: r, Body: body})
			p.bufferLock.Unlock()
			log.Info("Webhook request buffered")
			w.Write([]byte(""))
			return
		}
	} else {
		r.Host = fmt.Sprintf(p.newUrl, "vpavlin")
		//Switch or add OSO token
		r.Header["Authorization"] = []string{fmt.Sprintf("Bearer %s", p.GetToken(""))}
	}

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req = p.prepareRequest(req, r, body)
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		},
	}).ServeHTTP(w, r)
}

func (p *Proxy) GetUser(repo string) string {
	return strings.Split(repo, "/")[0]
}

func (p *Proxy) GetToken(token string) string {
	return p.token
}

func (p *Proxy) ProcessBuffer() {
	for {
		for name, rbs := range p.RequestBuffer {
			for i, rb := range *rbs {
				namespace := fmt.Sprintf("%s-jenkins", name)
				log.Info("Retrying request for ", namespace)
				if !p.OC.IsIdle(namespace, p.service) {
					req, reqErr := http.NewRequest("", "", nil)
					if reqErr != nil {
						log.Error("Request error ", reqErr)
						continue
					}
					req = p.prepareRequest(req, rb.Request, rb.Body)
					client := &http.Client{
						Transport: &http.Transport{
							Proxy: http.ProxyFromEnvironment,
							TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
						},
					}
					_, err := client.Do(req) 
					if err != nil {
						log.Error("Error: ", err)
					}

					p.bufferLock.Lock()
					if len(*rbs) > 1 {
						*rbs = append((*rbs)[:i], (*rbs)[i+1:]...)
					} else {
						*rbs = (*rbs)[:0]
					}
					p.bufferLock.Unlock()
				}
			}
		}
		time.Sleep(5*time.Second)
	}
}

func (p *Proxy) prepareRequest(dst *http.Request, src *http.Request, body []byte) *http.Request {
	dst.URL = src.URL
	dst.URL.Host = src.Host
	dst.URL.Scheme = "https" //FIXME
	dst.Host = src.Host
	dst.Method = src.Method

	for k, v := range src.Header {
		dst.Header[k] = v
	}
	dst.Header["Server"] = []string{"Webhook-Proxy"}

	dst.Body = ioutil.NopCloser(bytes.NewReader(body))
	
	return dst
}

type GHHookStruct struct {
	Repository struct {
		Name string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

func loadHookPayload(b []byte) *GHHookStruct {
	gh := &GHHookStruct{}
	err := json.Unmarshal(b, &gh)
	if err != nil {
		log.Error(err)
	}

	return gh
}