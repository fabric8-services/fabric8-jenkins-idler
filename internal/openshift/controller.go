package openshift

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/condition"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/configuration"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	pc "github.com/fabric8-services/fabric8-jenkins-proxy/clients"
	log "github.com/sirupsen/logrus"
)

const (
	availableCond = "Available"
)

type Controller interface {
	HandleBuild(o model.Object) (bool, error)
	HandleDeploymentConfig(dc model.DCObject) (bool, error)
	GetUsers() map[string]*model.User
	Run()
}

type OpenShiftController struct {
	Conditions       condition.Conditions
	Users            map[string]*model.User
	lock             *sync.Mutex
	checkIdleSleep   time.Duration
	FilterNamespaces []string
	openShiftClient  OpenShiftClient
	MaxUnIdleRetries int
	tenant           *pc.Tenant
	features         toggles.Features
}

func NewOpenShiftController(openShiftClient OpenShiftClient, t *pc.Tenant, features toggles.Features, config configuration.Configuration) Controller {
	controller := OpenShiftController{
		openShiftClient:  openShiftClient,
		Users:            make(map[string]*model.User),
		lock:             &sync.Mutex{},
		MaxUnIdleRetries: config.GetUnIdleRetry(),
		tenant:           t,
		features:         features,
	}

	controller.Conditions.Conditions = make(map[string]condition.Condition)

	//Add a Build condition
	bc := condition.NewBuildCondition(time.Duration(config.GetIdleAfter()) * time.Minute)
	controller.Conditions.Conditions["build"] = bc
	//Add a DeploymentConfig condition
	dcc := condition.NewDCCondition(time.Duration(config.GetIdleAfter()) * time.Minute)
	controller.Conditions.Conditions["DC"] = dcc
	//If we have access to Proxy, add User condition
	if len(config.GetProxyURL()) > 0 {
		log.Info("Adding 'user' condition")
		uc := condition.NewUserCondition(config.GetProxyURL(), time.Duration(config.GetIdleAfter())*time.Minute)
		controller.Conditions.Conditions["user"] = uc
	}

	return &controller
}

//HandleBuild processes new Build event collected from OpenShift and updates
//user structure with latest build info. NOTE: In most cases the only change in
//build object is stage timesstamp, which we don't care about, so this function
//just does couple comparisons and returns
func (oc *OpenShiftController) HandleBuild(o model.Object) (watched bool, err error) {
	watched = false

	ns := o.Object.Metadata.Namespace
	err = oc.checkNewUser(o.Object.Metadata.Namespace)
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
	if oc.isActive(&o.Object) {
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
		oc.Users[ns].ActiveBuild = &model.Build{Status: model.Status{Phase: "New"}}
		oc.lock.Unlock()
	}

	return
}

//HandleDeploymentConfig processes new DC event collected from OpenShift and updates
//user structure with info about the changes in DC. NOTE: This is important for cases
//like reset tenant and update tenant when DC is updated and Jenkins starts because
//of ConfigChange or manual intervention.
func (oc *OpenShiftController) HandleDeploymentConfig(dc model.DCObject) (watched bool, err error) {
	watched = false
	ns := dc.Object.Metadata.Namespace[:len(dc.Object.Metadata.Namespace)-len("-jenkins")]
	err = oc.checkNewUser(ns)
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

func (oc *OpenShiftController) GetUsers() map[string]*model.User {
	return oc.Users
}

//CheckIdle verifies the state of conditions and decides if we should idle/unidle
//and performs the required action if needed
func (oc *OpenShiftController) checkIdle(user *model.User) error {
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
		state, err := oc.openShiftClient.IsIdle(ns, "jenkins")
		if err != nil {
			return err
		}
		if state > model.JenkinsIdled {
			var n string
			var t time.Time
			if user.HasDone() {
				n = user.DoneBuild.Metadata.Name
				t = user.DoneBuild.Status.CompletionTimestamp.Time
			}
			log.Info(fmt.Sprintf("I'd like to idle jenkins for %s as last build finished at %s", user.Name, t))
			//Reset unidle retries and idle
			user.UnidleRetried = 0
			err := oc.openShiftClient.Idle(user.Name+"-jenkins", "jenkins") //FIXME - find better way to generate Jenkins namespace
			if err != nil {
				return err
			}

			user.AddJenkinsState(false, time.Now().UTC(), fmt.Sprintf("Jenkins Idled for %s, finished at %s", n, t))
		}
	} else { // UnIdle
		state, err := oc.openShiftClient.IsIdle(ns, "jenkins")
		if err != nil {
			return err
		}
		if state == model.JenkinsIdled {
			log.Debug("Potential unidling event")

			//Skip some retries,but check from time to time if things are fixed
			if user.UnidleRetried > oc.MaxUnIdleRetries && (user.UnidleRetried%oc.MaxUnIdleRetries != 0) {
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
			err := oc.openShiftClient.UnIdle(ns, "jenkins")
			if err != nil {
				return errors.New(fmt.Sprintf("Could not unidle Jenkins: %s", err))
			}
			user.AddJenkinsState(true, time.Now().UTC(), fmt.Sprintf("Jenkins Unidled for %s at %s", n, t))
		}
	}

	return nil
}

//CheckNewUser check existence of a user in the map, initialise if it does not exist
func (oc *OpenShiftController) checkNewUser(ns string) error {
	if _, exist := oc.Users[ns]; !exist {
		log.Debugf("New user %s", ns)
		state, err := oc.openShiftClient.IsIdle(ns+"-jenkins", "jenkins")
		if err != nil {
			return err
		}
		ti, err := oc.tenant.GetTenantInfoByNamespace(oc.openShiftClient.GetApiURL(), ns)
		if err != nil {
			return err
		}

		if ti.Meta.TotalCount > 1 {
			return fmt.Errorf("Could not add new user - Tenant service returned multiple items: %d", ti.Meta.TotalCount)
		} else if len(ti.Data) == 0 {
			return fmt.Errorf("Could not find tenant in cluster %s for namespace %s: %+v", oc.openShiftClient.GetApiURL(), ns, ti.Errors)
		}

		oc.lock.Lock()
		oc.Users[ns] = model.NewUser(ti.Data[0].Id, ns, (state == model.JenkinsRunning))
		oc.lock.Unlock()
		log.Debugf("Recorded new user %s", ns)
	} else {
		log.Debugf("User %s exists", ns)
	}

	return nil
}

//Run implements main loop of the application
func (oc *OpenShiftController) Run() {
	go oc.openShiftClient.WatchDeploymentConfigs("", "-jenkins", oc.HandleDeploymentConfig)
	go oc.openShiftClient.WatchBuilds("", "JenkinsPipeline", oc.HandleBuild)

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
				err = oc.checkIdle(u)
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
	err := json.Indent(&prettyJSON, data, "", "\t")
	if err != nil {
		log.Println("JSON parse error: ", err)
		return
	}

	log.Println(string(prettyJSON.Bytes()))
}

//IsActive returns true ifa build phase suggests a build is active.
//It returns false otherwise.
func (oc *OpenShiftController) isActive(b *model.Build) bool {
	return model.Phases[b.Status.Phase] == 1
}
