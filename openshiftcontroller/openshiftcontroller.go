package openshiftcontroller

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"time"
	"sync"

	log "github.com/sirupsen/logrus"
)

var Phases = map[string]int {
	"Finished": 0,
	"Complete": 0,
	"Failed": 0,
	"Cancelled": 0,
	"Pending": 1,
	"New": 1,
	"Running": 1,
}

type OpenShiftController struct {
	OpenShiftControllerI
	apiURL string
	token string
	Phases map[string]int
	Conditions Conditions
	Users map[string]*User
	lock *sync.Mutex
	groupLock *sync.Mutex
	Groups []*[]string
	groupSleep time.Duration
	FilterNamespaces []string
}

type OpenShiftControllerI interface {
	Login(token string) bool;
	Idle(namespace string, service string) bool;
	Run()
	IsIdle(namespace string, service string) bool;
}

type DeploymentConfig struct {
	Metadata Metadata `json:"metadata"`
	Status DCStatus `json:"Status"`
}

type DCStatus struct {
	Replicas int
	ReadyReplicas int
}

type Scale struct {
	Kind string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
	Metadata Metadata `json:"metadata"`
	Spec struct {
		Replicas int `json:"replicas"`
	} `json:"spec"`
}

func NewOpenShiftController(apiURL string, token string, nGroups int, idleAfter int, filter []string) *OpenShiftController {
	oc := &OpenShiftController{}
	oc.apiURL = apiURL
	oc.token = token
	oc.Conditions.Conditions = make(map[string]ConditionI)
	oc.Users = make(map[string]*User)
	oc.lock = &sync.Mutex{}
	oc.groupLock = &sync.Mutex{}
	oc.Groups = make([]*[]string, nGroups)
	oc.groupSleep = 10*time.Second
	oc.FilterNamespaces = filter
	
	oc.LoadProjects()

	bc := NewBuildCondition(time.Duration(idleAfter)*time.Minute)
	oc.Conditions.Conditions["build"] = bc

	return oc
}

func (oc *OpenShiftController) Idle(namespace string, service string) bool {
	log.Info("Idling "+service+" in "+namespace)
	log.Info("oc idle "+service+" -n "+namespace)
	return true
}

func (oc *OpenShiftController) UnIdle(namespace string, service string) bool {
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
		log.Error(err)
		return false
	}
	br := ioutil.NopCloser(bytes.NewReader(body))
	req, err := http.NewRequest("PUT", oc.constructRequest(namespace, fmt.Sprintf("deploymentconfigs/%s/scale", service), false), br) //FIXME
	if err != nil {
		log.Error(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oc.token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return false
	}
	defer resp.Body.Close()
	b, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		log.Error(err)
	}
	log.Warn(string(b))

	return true
}

func (oc *OpenShiftController) IsIdle(namespace string, service string) bool {
	url := oc.constructRequest(namespace, "deploymentconfigs/"+service, false)
	resp := oc.get(url, oc.token)
	defer resp.Body.Close()
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var dc DeploymentConfig
	json.Unmarshal(body, &dc)
	log.Info(dc)
	/*if len(dc.Metadata.Annotations.IdledAt) > 0 {
		return true
	} */

	if dc.Status.ReadyReplicas == 0 {
		return true
	}

	return false
}

func (oc *OpenShiftController) HandleBuildInfo(b *Build, spawnCheckIdle bool) {
	namespace := b.Metadata.Namespace
	if _, exist := oc.Users[namespace]; !exist {
		oc.lock.Lock()
		oc.Users[namespace] = NewUser(namespace)
		oc.lock.Unlock()
		if spawnCheckIdle {
			log.Info("Spawning Idling Checker for ", namespace)
			user := oc.Users[namespace]
			go oc.CheckIdle(user)
		}
	} 

	if Phases[b.Status.Phase] == 0 {
		oc.Users[namespace].DoneBuilds[b.Metadata.Name] = *b
		if _, exist := oc.Users[namespace].ActiveBuilds[b.Metadata.Name]; exist {
			delete(oc.Users[namespace].ActiveBuilds, b.Metadata.Name)
		}
	} else {
		oc.Users[namespace].ActiveBuilds[b.Metadata.Name] = *b
	}
}

func (oc *OpenShiftController) CheckIdle(user *User) {
	oc.lock.Lock()
	eval := oc.Conditions.Eval(user)
	oc.lock.Unlock()
	if eval {
		b := user.LastDone()
		if user.JenkinsStateList[len(user.JenkinsStateList)-1].Running {
			log.Warn(fmt.Sprintf("I'd like to idle jenkins for %s as last build finished at %s", user.Name,
				b.Status.CompletionTimestamp.Time))
			oc.Idle(user.Name+"-jenkins", "jenkins")
			user.AddJenkinsState(false, time.Now().UTC(), fmt.Sprintf("Jenkins Idled for %s, finished at %s", b.Metadata.Name, b.Status.CompletionTimestamp.Time))
			fmt.Printf("%+v\n", user.JenkinsStateList)
		}
	} else {
		b := user.LastDone()
		if !user.JenkinsStateList[len(user.JenkinsStateList)-1].Running {
			oc.UnIdle(user.Name+"-jenkins", "jenkins")
			user.AddJenkinsState(true, time.Now().UTC(), fmt.Sprintf("Jenkins Unidled for %s at %s", b.Metadata.Name, time.Now().UTC()))
		}
	}
}

func (oc *OpenShiftController) processBuilds(namespaces []string) {
	for _, n := range namespaces {
		if _, exist := oc.Users[n]; !exist {
			oc.lock.Lock()
			oc.Users[n] = NewUser(n)
			oc.lock.Unlock()
		}
		//log.Info("Getting builds for ", n)
		url := oc.constructRequest(n, "builds", false)
		resp := oc.get(url, oc.token)
		if resp != nil {
			defer resp.Body.Close()

			var bl BuildList

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
					log.Fatal(err)
			}

			json.Unmarshal(body, &bl)

			for _, b := range bl.Items {
				oc.HandleBuildInfo(&b, false)
			}
		}
	}
}

func (oc *OpenShiftController) ServeJenkinsStates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(oc.Users)
	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprintf(w, "{'msg': 'Could not serialize users'}")
	}

	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprintf(w, "{'msg': 'Could not serialize users'}")
	}
	
	//fmt.Fprint(w, us)
}

func (oc *OpenShiftController) Run(groupNumber int) {
	for {
		log.Info("Checking group #", groupNumber)
		oc.processBuilds(*oc.Groups[groupNumber])

		for _, n := range *oc.Groups[groupNumber] {
			oc.CheckIdle(oc.Users[n])
		}
		time.Sleep(oc.groupSleep)
	}
}

func (oc *OpenShiftController) LoadProjects() []string {
	url := oc.constructRequest("", "projects", false)
	resp := oc.get(url, oc.token)
	var projects []string
	if resp == nil {
		projects = []string{}
	} else {
		defer resp.Body.Close()
		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			log.Error(readErr) 
		}
		projects = ProcessProjects(body, oc.FilterNamespaces)
	}

	g := SplitGroups(projects, oc.Groups)
	oc.groupLock.Lock()
	oc.Groups = g
	oc.groupLock.Unlock()
	fmt.Printf("%+v\n", oc.Groups)

	return projects
}

func (oc *OpenShiftController) DownloadProjects() {
	log.Info("Updating namespace list")
	url := oc.constructRequest("", "projects", false)
	resp := oc.get(url, oc.token)
	var projects []string
	if resp != nil {
		defer resp.Body.Close()
		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}
		projects = ProcessProjects(body, oc.FilterNamespaces)
	} else {
		projects = []string{}
	}

	g, err := UpdateProjects(oc.Groups, projects)
	if err != nil {
		log.Error(err)
	} else {
		oc.groupLock.Lock()
		oc.Groups = g
		oc.groupLock.Unlock()
	}
}


func (oc *OpenShiftController) constructRequest(namespace string, command string, watch bool) string {
	url := "https://"+oc.apiURL+"/oapi/v1"
	if watch {
		url = fmt.Sprintf("%s/%s", url, "watch")
	}
	if len(namespace) > 0 {
		url = fmt.Sprintf("%s/%s/%s", url, "namespaces", namespace)
	}

	url = fmt.Sprintf("%s/%s", url, command)

	//log.Info("Generated URL: ", url)
	return url
}

func (oc *OpenShiftController) get(url string, token string) (resp *http.Response) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err = client.Do(req)
	if err != nil {
		log.Error("Could not perform the request: ", err)
		return
	}
	if resp.StatusCode != 200 {
		log.Error("Got status  ", resp.Status)
	}

	return resp
}

func (oc *OpenShiftController) prettyPrint(data []byte) {
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, data, "", "\t")
	if error != nil {
			log.Println("JSON parse error: ", error)
			return
	}

	log.Println(string(prettyJSON.Bytes()))	
}