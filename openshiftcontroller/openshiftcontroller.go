package openshiftcontroller

import (
	"strconv"
	"errors"
	"fmt"
	"encoding/json"
	"bytes"
	"time"
	"sync"

	log "github.com/sirupsen/logrus"
	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
)

const (
	loadRetrySleep   = 10
	availableCond    = "Available"
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
	
	var err error
	for { //FIXME
		_, err = oc.LoadProjects()
		if err != nil {
			log.Error(err)
			time.Sleep(loadRetrySleep*time.Second)
		} else {
			break
		}
	}

	bc := NewBuildCondition(time.Duration(idleAfter)*time.Minute)
	oc.Conditions.Conditions["build"] = bc
	dcc := NewDCCondition(time.Duration(idleAfter)*time.Minute)
	oc.Conditions.Conditions["DC"] = dcc
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
	eval, condStates := oc.Conditions.Eval(user)
	oc.lock.Unlock()
	cs, _ := json.Marshal(condStates) //Ignore errors
	log.Infof("Conditions: %b = %s", eval, string(cs))
	if eval {
		state, err := oc.o.IsIdle(ns, "jenkins")
		if err != nil {
			return err
		}
		if state > ic.JenkinsIdled {
			var n string
			var t time.Time
			if user.HasDone() {
				n = user.DoneBuild.Metadata.Name
				t = user.DoneBuild.Status.CompletionTimestamp.Time
			}
			log.Info(fmt.Sprintf("I'd like to idle jenkins for %s as last build finished at %s", user.Name,	t))
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
		if state == ic.JenkinsIdled {
			log.Debug("Potential unidling event")
			if user.UnidleRetried > oc.MaxUnidleRetries && (user.UnidleRetried % oc.MaxUnidleRetries != 0) { //Skip some retries,but check from time to time if things are fixed
				user.UnidleRetried++
				 log.Debug(fmt.Sprintf("Skipping unidle for %s, too many retries", user.Name))
				 return nil
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

func (oc *OpenShiftController) HandleBuild(o ic.Object) (err error) {
	found := false
	ns := o.Object.Metadata.Namespace
	if len(oc.FilterNamespaces) > 0 {
		for _, n := range oc.FilterNamespaces {
			if ns == n {
				found = true
				break
			}
		}
	} else {
		found = true
	}

	if !found {
		log.Infof("Throwing event away: %s (%s)", o.Object.Metadata.Name, o.Object.Metadata.Namespace)
		return
	}

	err = oc.CheckNewUser(o.Object.Metadata.Namespace)
	if err != nil {
		return
	}
	log.Infof("Processing %s", o.Object.Metadata.Name)
	if IsActive(&o.Object) {
		lastActive := oc.Users[ns].ActiveBuild
		if lastActive.Status.Phase != o.Object.Status.Phase || lastActive.Metadata.Annotations.BuildNumber != o.Object.Metadata.Annotations.BuildNumber {
			oc.lock.Lock()
			*oc.Users[ns].ActiveBuild = o.Object
			oc.lock.Unlock()
		}
	} else {
		lastDone := oc.Users[ns].DoneBuild
		if lastDone.Status.Phase != o.Object.Status.Phase || lastDone.Metadata.Annotations.BuildNumber != o.Object.Metadata.Annotations.BuildNumber {
			oc.lock.Lock()
			*oc.Users[ns].DoneBuild = o.Object
			oc.lock.Unlock()
		}
	}

	if oc.Users[ns].ActiveBuild.Metadata.Annotations.BuildNumber == oc.Users[ns].DoneBuild.Metadata.Annotations.BuildNumber {
		oc.lock.Lock()
		oc.Users[ns].ActiveBuild = &ic.Build{Status: ic.Status{Phase: "New"}}
		oc.lock.Unlock()
	}

	return
}

func (oc *OpenShiftController) HandleDeploymentConfig(dc ic.DCObject) (err error) {
	found := false
	ns := dc.Object.Metadata.Namespace[:len(dc.Object.Metadata.Namespace)-len("-jenkins")]
	for _, n := range oc.FilterNamespaces {
		if ns == n {
			found = true
			break
		}
	}

	if len(oc.FilterNamespaces) == 0 {
		found = true
	}

	if !found {
		log.Infof("Throwing event away: %s (%s)", dc.Object.Metadata.Name, dc.Object.Metadata.Namespace)
		return
	}

	err = oc.CheckNewUser(ns)
	if err != nil {
		return
	}

	c, err := dc.Object.Status.GetByType(availableCond)
	if err != nil {
		return
	}

	if (dc.Object.Metadata.Generation != dc.Object.Status.ObservedGeneration && dc.Object.Spec.Replicas > 0) || dc.Object.Status.UnavailableReplicas > 0 {
		oc.lock.Lock()
		oc.Users[ns].JenkinsLastUpdate = time.Now().UTC()
		oc.lock.Unlock()
	}

	status, err := strconv.ParseBool(c.Status)
	if err != nil {
		return
	}
	if status == true {
		oc.lock.Lock()
		oc.Users[ns].JenkinsLastUpdate = c.LastUpdateTime
		oc.lock.Unlock()
	}

	return
}

func (oc *OpenShiftController) CheckNewUser(ns string) (err error) {
	if _, exist := oc.Users[ns]; !exist {
		state, err := oc.o.IsIdle(ns+"-jenkins", "jenkins")
		if err != nil {
			return err
		}
		oc.lock.Lock()
		oc.Users[ns] = NewUser(ns, (state == ic.JenkinsRunning))
		oc.lock.Unlock()
	}

	return
}

func (oc *OpenShiftController) processBuilds(namespaces []string) (err error) {
	for _, n := range namespaces {
		err := oc.CheckNewUser(n)
		if err != nil {
			return err
		}
		//log.Info("Getting builds for ", n)

		bl, err := oc.o.GetBuilds(n)
		if err != nil {
			log.Error("Could not load builds: ", err)
			continue
		}

		lastActive := oc.Users[n].ActiveBuild
		lastDone := oc.Users[n].DoneBuild
		for i, _ := range bl.Items {
			if IsActive(&bl.Items[i]) {
				lastActive, err = GetLastBuild(lastActive, &bl.Items[i])
			} else {
				lastDone, err = GetLastBuild(lastDone, &bl.Items[i])
			}
			if err != nil {
				log.Error(err)
			}
		}
		oc.lock.Lock()
		*oc.Users[n].ActiveBuild = *lastActive
		*oc.Users[n].DoneBuild = *lastDone
		oc.lock.Unlock()
	}

	return
}

func (oc *OpenShiftController) Run(groupNumber int, watch bool) {
	if watch {
		//FIXME
		go func() {
			for {
				for _, u := range oc.Users {
					err := oc.CheckIdle(u)
					if err != nil {
						log.Errorf("Could not check idling for %s: %s", u.Name, err)
					}
				}
				time.Sleep(oc.groupSleep)
			}
		}()
		go oc.o.WatchDeploymentConfigs("", "-jenkins", oc.HandleDeploymentConfig)
		oc.o.WatchBuilds("", "JenkinsPipeline", oc.HandleBuild)
		return
	}
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