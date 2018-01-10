package openshiftcontroller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	pc "github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	log "github.com/sirupsen/logrus"
)

const (
	loadRetrySleep = 10
	availableCond  = "Available"
)

type OpenShiftController struct {
	Phases           map[string]int
	Conditions       Conditions
	Users            map[string]*User
	lock             *sync.Mutex
	checkIdleSleep   time.Duration
	FilterNamespaces []string
	o                *ic.OpenShift
	MaxUnidleRetries int
	tenant           *pc.Tenant
	features         toggles.Features
}

func NewOpenShiftController(o *ic.OpenShift, t *pc.Tenant, idleAfter int, filter []string, proxyURL string, maxUnidleRetries int, features toggles.Features) *OpenShiftController {
	oc := &OpenShiftController{
		o:                o,
		Users:            make(map[string]*User),
		lock:             &sync.Mutex{},
		FilterNamespaces: filter,
		MaxUnidleRetries: maxUnidleRetries,
		tenant:           t,
		features:         features,
	}

	oc.Conditions.Conditions = make(map[string]Condition)

	//Add a Build condition
	bc := NewBuildCondition(time.Duration(idleAfter) * time.Minute)
	oc.Conditions.Conditions["build"] = bc
	//Add a DeploymentConfig condition
	dcc := NewDCCondition(time.Duration(idleAfter) * time.Minute)
	oc.Conditions.Conditions["DC"] = dcc
	//If we have access to Proxy, add User condition
	if len(proxyURL) > 0 {
		log.Info("Adding 'user' condition")
		uc := NewUserCondition(proxyURL, time.Duration(idleAfter)*time.Minute)
		oc.Conditions.Conditions["user"] = uc
	}

	return oc
}

//CheckIdle verifies the state of conditions and decides if we should idle/unidle
//and performs the required action if needed
func (oc *OpenShiftController) CheckIdle(user *User) error {
	if user == nil {
		return errors.New("Empty user")
	}
	ns := user.Name + "-jenkins"
	oc.lock.Lock()
	eval, condStates := oc.Conditions.Eval(user)
	oc.lock.Unlock()
	cs, _ := json.Marshal(condStates) //Ignore errors
	log.Debugf("Conditions: %b = %s", eval, string(cs))
	//Eval == true -> do Idle, Eval == false -> do Unidle
	if eval {
		//Check if Jenkins is running
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
			log.Info(fmt.Sprintf("I'd like to idle jenkins for %s as last build finished at %s", user.Name, t))
			//Reset unidle retries and idle
			user.UnidleRetried = 0
			err := oc.o.Idle(user.Name+"-jenkins", "jenkins") //FIXME - find better way to generate Jenkins namespace
			if err != nil {
				return err
			}

			user.AddJenkinsState(false, time.Now().UTC(), fmt.Sprintf("Jenkins Idled for %s, finished at %s", n, t))
		}
	} else { //Unidle
		state, err := oc.o.IsIdle(ns, "jenkins")
		if err != nil {
			return err
		}
		if state == ic.JenkinsIdled {
			log.Debug("Potential unidling event")

			//Skip some retries,but check from time to time if things are fixed
			if user.UnidleRetried > oc.MaxUnidleRetries && (user.UnidleRetried%oc.MaxUnidleRetries != 0) {
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
			//Inc unidle retries
			user.UnidleRetried++
			err := oc.o.UnIdle(ns, "jenkins")
			if err != nil {
				return errors.New(fmt.Sprintf("Could not unidle Jenkins: %s", err))
			}
			user.AddJenkinsState(true, time.Now().UTC(), fmt.Sprintf("Jenkins Unidled for %s at %s", n, t))
		}
	}

	return nil
}

//HandleBuild processes new Build event collected from OpenShift and updates
//user structure with latest build info. NOTE: In most cases the only change in
//build object is stage timesstamp, which we don't care about, so this function
//just does couple comparisons and returns
func (oc *OpenShiftController) HandleBuild(o ic.Object) (watched bool, err error) {
	watched = false

	ns := o.Object.Metadata.Namespace
	err = oc.CheckNewUser(o.Object.Metadata.Namespace)
	if err != nil {
		return
	}

	log.Debugf("Checking if idler is enabled for %s (%s)", ns, oc.Users[ns].ID)
	//Filter for configured namespaces, FIXME: Use toggle service instead
	enabled, err := oc.features.IsIdlerEnabled(oc.Users[ns].ID)
	if err != nil {
		return
	}

	log.Debugf("Result of toggle check for %s: %t", ns, enabled)
	if enabled {
		log.Infof("Idler enabled for %s", ns)
		watched = true
	} else if len(oc.FilterNamespaces) > 0 {
		log.Debug("Filtering namespaces")
		for _, n := range oc.FilterNamespaces {
			if ns == n {
				watched = true
				break
			}
		}
	}

	if !watched {
		log.Infof("Throwing event away: %s (%s)", o.Object.Metadata.Name, o.Object.Metadata.Namespace)
		return
	}

	log.Infof("Processing %s", ns)
	if IsActive(&o.Object) {
		lastActive := oc.Users[ns].ActiveBuild
		if lastActive.Status.Phase != o.Object.Status.Phase || lastActive.Metadata.Name != o.Object.Metadata.Name {
			oc.lock.Lock()
			*oc.Users[ns].ActiveBuild = o.Object
			oc.lock.Unlock()
		}
	} else {
		lastDone := oc.Users[ns].DoneBuild
		if lastDone.Status.Phase != o.Object.Status.Phase || lastDone.Metadata.Name != o.Object.Metadata.Name {
			oc.lock.Lock()
			*oc.Users[ns].DoneBuild = o.Object
			oc.lock.Unlock()
		}
	}

	//If we have same build name (space name + build number) in Active and Done build reference, it means last event was transition of an Active build into
	//Done build, we need to clean up the Active build ref
	if oc.Users[ns].ActiveBuild.Metadata.Name == oc.Users[ns].DoneBuild.Metadata.Name {
		log.Infof("Active and Done builds for %s are the same (%s), claning active builds", ns, oc.Users[ns].ActiveBuild.Metadata.Name)
		oc.lock.Lock()
		oc.Users[ns].ActiveBuild = &ic.Build{Status: ic.Status{Phase: "New"}}
		oc.lock.Unlock()
	}

	return
}

//HandleDeploymentConfig processes new DC event collected from OpenShift and updates
//user structure with info about the changes in DC. NOTE: This is important for cases
//like reset tenant and update tenant when DC is updated and Jenkins starts because
//of ConfigChange or manual intervention.
func (oc *OpenShiftController) HandleDeploymentConfig(dc ic.DCObject) (watched bool, err error) {
	watched = false
	ns := dc.Object.Metadata.Namespace[:len(dc.Object.Metadata.Namespace)-len("-jenkins")]
	err = oc.CheckNewUser(ns)
	if err != nil {
		return
	}
	log.Debugf("Checking if user %s (%s) is enabled", oc.Users[ns].Name, oc.Users[ns].ID)

	enabled, err := oc.features.IsIdlerEnabled(oc.Users[ns].ID)
	if err != nil {
		return
	}

	if enabled {
		log.Infof("Idler enabled for %s", ns)
		watched = true
	} else if len(oc.FilterNamespaces) > 0 {
		for _, n := range oc.FilterNamespaces {
			if ns == n {
				watched = true
				break
			}
		}
	}

	if !watched {
		log.Infof("Throwing event away: %s (%s)", dc.Object.Metadata.Name, dc.Object.Metadata.Namespace)
		return
	}

	c, err := dc.Object.Status.GetByType(availableCond)
	if err != nil {
		return
	}

	//This is either a new version of DC or we existing version waiting to come up;FIXME: Verify if we need Generation vs. ObservedGeneration
	if (dc.Object.Metadata.Generation != dc.Object.Status.ObservedGeneration && dc.Object.Spec.Replicas > 0) || dc.Object.Status.UnavailableReplicas > 0 {
		oc.lock.Lock()
		oc.Users[ns].JenkinsLastUpdate = time.Now().UTC()
		oc.lock.Unlock()
	}

	//Also check if the event means that Jenkins just started (OS AvailableCondition.Status == true) and update time
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

//CheckNewUser check existance of a user in the map, initialise if it does not exist
func (oc *OpenShiftController) CheckNewUser(ns string) (err error) {
	if _, exist := oc.Users[ns]; !exist {
		log.Debugf("New user %s", ns)
		state, err := oc.o.IsIdle(ns+"-jenkins", "jenkins")
		if err != nil {
			return err
		}
		ti, err := oc.tenant.GetTenantInfoByNamespace(oc.o.GetApiURL(), ns)
		if err != nil {
			return err
		}

		if ti.Meta.TotalCount > 1 {
			return fmt.Errorf("Could not add new user - Tenant service returned multiple items: %d", ti.Meta.TotalCount)
		} else if len(ti.Data) == 0 {
			return fmt.Errorf("Could not find tenant in cluster %s for namespace %s: %+v", oc.o.GetApiURL(), ns, ti.Errors)
		}

		oc.lock.Lock()
		oc.Users[ns] = NewUser(ti.Data[0].Id, ns, (state == ic.JenkinsRunning))
		oc.lock.Unlock()
		log.Debugf("Recorded new user %s", ns)
	} else {
		log.Debugf("User %s exists", ns)
	}

	return
}

//Run implements main loop of the application
func (oc *OpenShiftController) Run() {
	go oc.o.WatchDeploymentConfigs("", "-jenkins", oc.HandleDeploymentConfig)
	go oc.o.WatchBuilds("", "JenkinsPipeline", oc.HandleBuild)

	//FIXME - this looks ugly
	go func() {
		for {
			//For each user we know about, check if there is any action needed
			for _, u := range oc.Users {
				enabled, err := oc.features.IsIdlerEnabled(u.ID)
				if err != nil {
					log.Error("Error checking for idler feature", err)
					continue
				}
				if !enabled {
					log.Debugf("Skipping check for %s.", u.Name)
					continue
				}
				err = oc.CheckIdle(u)
				if err != nil {
					log.Errorf("Could not check idling for %s: %s", u.Name, err)
				}
			}
			time.Sleep(oc.checkIdleSleep)
		}
	}()
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
