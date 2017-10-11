package openshiftcontroller

import (
	"bufio"
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"strings"
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

func NewOpenShiftController(apiURL string, token string) *OpenShiftController {
	oc := &OpenShiftController{}
	oc.apiURL = apiURL
	oc.token = token
	oc.Conditions.Conditions = make(map[string]ConditionI)
	oc.Users = make(map[string]*User)
	oc.lock = &sync.Mutex{}

	//oc.Conditions = append(oc.Conditions, &BuildCondition{})
	//oc.Conditions = append(oc.Conditions, &UserCondition{})
	bc := NewBuildCondition(1*time.Minute)
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
	return true
}

func (oc *OpenShiftController) IsIdle(namespace string, service string) bool {
	url := oc.constructRequest(namespace, "deploymentconfigs/"+service, false)
	resp := oc.get(url, oc.token)
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var dc DeploymentConfig
	json.Unmarshal(body, &dc)
	log.Info(dc)
	if len(dc.Metadata.Annotations.IdledAt) > 0 {
		return true
	} 

	if dc.Status.ReadyReplicas == 0 {
		return true
	}

	return false
}

func (oc *OpenShiftController) HandleBuildInfo(b *Build, spawnCheckIdle bool) {
	namespace := b.Metadata.Namespace
	if _, exist := oc.Users[namespace]; !exist {
		oc.Users[namespace] = &User{
			Name: namespace, 
			ActiveBuilds: make(map[string]Build), 
			DoneBuilds: make(map[string]Build), 
			JenkinsStateList: []JenkinsState{JenkinsState{true, time.Now().UTC(), "init"}},
		}
		user := oc.Users[namespace]
		if spawnCheckIdle {
			log.Info("Spawning Idling Checker for ", namespace)
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
	for {
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
				user.AddJenkinsState(true, time.Now().UTC(), fmt.Sprintf("Jenkins Unidled for %s, started at %s", b.Metadata.Name, b.Status.StartTimestamp.Time))
			}
		}

		time.Sleep(5*time.Second)
	}
}

func (oc *OpenShiftController) initBuilds(namespaces []string) {
	for _, n := range namespaces {
		url := oc.constructRequest(n, "builds", false)
		resp := oc.get(url, oc.token)
	
		defer resp.Body.Close()

		var bl BuildList

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
				panic(err.Error())
		}

		json.Unmarshal(body, &bl)

		for _, b := range bl.Items {
			oc.HandleBuildInfo(&b, false)
		}
	}
}

func (oc *OpenShiftController) ServeJenkinsStates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(oc.Users["vpavlin"].JenkinsStateList)
	log.Info(fmt.Sprintf("Items in Jenkins State list: %d", len(oc.Users["vpavlin"].JenkinsStateList)))
	if err != nil {
		log.Error("Could not serialize users")
		fmt.Fprintf(w, "{'msg': 'Could not serialize users'}")
	}
	
	//fmt.Fprint(w, us)
}

func (oc *OpenShiftController) Run(namespaces []string) {
	oc.initBuilds(namespaces)

	http.HandleFunc("/builds", oc.ServeJenkinsStates)
	go http.ListenAndServe(":8080", nil)

	for _, u1 := range oc.Users {
		go oc.CheckIdle(u1)
	}

	n := ""
	if len(namespaces) == 1 {
		n = namespaces[0]
	}
	url := oc.constructRequest(n, "builds", true)
	client := &http.Client{}
	
	var x []Build

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", oc.token))
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
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

				
				var o Object
			
				err = json.Unmarshal(line, &o)
				if err!=nil {
					if strings.HasPrefix(string(line), "This request caused apisever to panic") {
						log.WithField("error", string(line)).Warning("Communication with server failed")
						break;
					}
					log.Fatal("Failed to Unmarshal: ", err)
				}
				//log.Info(o.Object.Metadata.Name)
				x = append(x, o.Object)

				elemIn := false
				for _, nc := range namespaces {
				 if nc == o.Object.Metadata.Namespace {
					elemIn = true
					break
				 }
				}

				if !elemIn {
					continue
				}

				oc.lock.Lock()
				oc.HandleBuildInfo(&o.Object, true)
				oc.lock.Unlock()

				u := oc.Users[o.Object.Metadata.Namespace]
				fmt.Printf("Handled event for user %s\n", o.Object.Metadata.Namespace)

				fmt.Printf("# of Active Builds: %d\n", len(u.ActiveBuilds))
				fmt.Printf("# of Done Builds: %d\n", len(u.DoneBuilds))
				
				
				fmt.Printf("Event summary: Build %s -> %s, %s/%s\n", o.Object.Metadata.Name, o.Object.Status.Phase, o.Object.Status.StartTimestamp, o.Object.Status.CompletionTimestamp) 

		}
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

	log.Info("Generated URL: ", url)
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
	if resp.StatusCode != 200 {
		log.Error("Got status  ", resp.Status)
	}
	if err != nil {
		log.Error(err)
		return
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