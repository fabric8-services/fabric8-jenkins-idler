package openshiftcontroller

import (
	"errors"
	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	"fmt"
	"encoding/json"
	"bytes"
	"time"
	"sync"

	log "github.com/sirupsen/logrus"
)

type OpenShiftController struct {
	Phases map[string]int
	Conditions Conditions
	Users map[string]*User
	lock *sync.Mutex
	groupLock *sync.Mutex
	Groups []*[]string
	groupSleep time.Duration
	FilterNamespaces []string
	o ic.OpenShift
	MaxUnidleRetries int
}

func NewOpenShiftController(o ic.OpenShift, nGroups int, idleAfter int, filter []string, proxyURL string, maxUnidleRetries int) *OpenShiftController {
	oc := &OpenShiftController{
		o: o,
	}
	oc.Conditions.Conditions = make(map[string]ConditionI)
	oc.Users = make(map[string]*User)
	oc.lock = &sync.Mutex{}
	oc.groupLock = &sync.Mutex{}
	oc.Groups = make([]*[]string, nGroups)
	oc.groupSleep = 10*time.Second
	oc.FilterNamespaces = filter
	oc.MaxUnidleRetries = maxUnidleRetries
	
	oc.LoadProjects()

	bc := NewBuildCondition(time.Duration(idleAfter)*time.Minute)
	oc.Conditions.Conditions["build"] = bc
	if len(proxyURL) > 0 {
		log.Info("Adding 'user' condition")
		uc := NewUserCondition(proxyURL, time.Duration(idleAfter)*time.Minute)
		oc.Conditions.Conditions["user"] = uc
	}

	return oc
}

func (oc *OpenShiftController) CheckIdle(user *User) (error) {
	if user == nil {
		return errors.New("Empty user")
	}
	ns := user.Name+"-jenkins"
	oc.lock.Lock()
	eval := oc.Conditions.Eval(user)
	oc.lock.Unlock()
	if eval {
		state, err := oc.o.IsIdle(ns, "jenkins")
		if err != nil {
			return err
		}
		if state > ic.JenkinsStates["Idle"] {
			var n string
			var t time.Time
			if user.HasDone() {
				n = user.DoneBuild.Metadata.Name
				t = user.DoneBuild.Status.CompletionTimestamp.Time
			}
			log.Warn(fmt.Sprintf("I'd like to idle jenkins for %s as last build finished at %s", user.Name,	t))
			user.UnidleRetried = 0
			err := oc.o.Idle(user.Name+"-jenkins", "jenkins")
			if err != nil {
				return err
			}

			user.AddJenkinsState(false, time.Now().UTC(), fmt.Sprintf("Jenkins Idled for %s, finished at %s", n, t))
		}
	} else {
		state, err := oc.o.IsIdle(ns, "jenkins")
		if err != nil {
			return err
		}
		if state == ic.JenkinsStates["Idle"] {
			if user.UnidleRetried > oc.MaxUnidleRetries {
				return errors.New(fmt.Sprintf("Skipping unidle for %s, too many retries", user.Name))
			}
			var n string
			var t time.Time
			if user.HasActive() {
				n = user.ActiveBuild.Metadata.Name
				t = user.ActiveBuild.Status.CompletionTimestamp.Time
			}
			err := oc.o.UnIdle(ns, "jenkins")
			if err != nil {
				return errors.New(fmt.Sprintf("Could not unidle Jenkins: %s", err))
			}
			user.UnidleRetried++
			user.AddJenkinsState(true, time.Now().UTC(), fmt.Sprintf("Jenkins Unidled for %s at %s", n, t))
		}
	}

	return nil
}

func (oc *OpenShiftController) processBuilds(namespaces []string) {
	for _, n := range namespaces {
		if _, exist := oc.Users[n]; !exist {
			state, err := oc.o.IsIdle(n+"-jenkins", "jenkins")
			if err != nil {
				log.Error(err)
				continue
			}
			oc.lock.Lock()
			oc.Users[n] = NewUser(n, (state == ic.JenkinsStates["Running"]))
			oc.lock.Unlock()
		}
		//log.Info("Getting builds for ", n)

		bl, err := oc.o.GetBuilds(n)
		if err != nil {
			log.Error("Could not load builds: ", err)
			continue
		}

		var lastActive *ic.Build
		var lastDone *ic.Build

		lastActive = nil
		lastDone = nil
		for i, _ := range bl.Items {
			if IsActive(&bl.Items[i]) {
				lastActive, _ = GetLastBuild(lastActive, &bl.Items[i])
			} else {
				lastDone, _ = GetLastBuild(lastDone, &bl.Items[i])
			}
		}
		oc.lock.Lock()
		if lastActive != nil {
			*oc.Users[n].ActiveBuild = *lastActive
		}
		if lastDone != nil {
			*oc.Users[n].DoneBuild = *lastDone
		}
		oc.lock.Unlock()
	}
}

func (oc *OpenShiftController) Run(groupNumber int) {
	for {
		log.Info("Checking group #", groupNumber)
		oc.processBuilds(*oc.Groups[groupNumber])

		for _, n := range *oc.Groups[groupNumber] {
			err := oc.CheckIdle(oc.Users[n])
			if err != nil {
				log.Error(n)
				log.Error(err)
			}
		}
		time.Sleep(oc.groupSleep)
	}
}

func (oc *OpenShiftController) LoadProjects() (projects[] string, err error) {
	projects, err = oc.o.GetProjects()
	if err != nil {
		return
	}
	projects = FilterProjects(projects, oc.FilterNamespaces)
	
	g := SplitGroups(projects, oc.Groups)
	oc.groupLock.Lock()
	oc.Groups = g
	oc.groupLock.Unlock()
	fmt.Printf("%+v\n", oc.Groups)

	return
}

func (oc *OpenShiftController) DownloadProjects() (err error) {
	projects, err := oc.o.GetProjects()
	if err != nil {
		return err
	}
	projects = FilterProjects(projects, oc.FilterNamespaces)

	g, err := UpdateProjects(oc.Groups, projects)
	if err != nil {
		return
	} 

	oc.groupLock.Lock()
	oc.Groups = g
	oc.groupLock.Unlock()

	return
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