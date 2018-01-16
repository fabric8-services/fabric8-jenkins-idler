package condition

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	proxyAPI "github.com/fabric8-services/fabric8-jenkins-proxy/api"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	log "github.com/sirupsen/logrus"
)

//Condition is a minimal interface a condition needs to implement
type Condition interface {
	//Return true if the condition is true for a given object
	IsTrueFor(object interface{}) (bool, error)
}

type Conditions struct {
	Conditions map[string]Condition
}

//Eval evaluates a list of conditions for a given object. It returns false if
//any of the conditions evals as false
func (c *Conditions) Eval(o interface{}) (result bool, condStates map[string]bool) {
	result = true
	condStates = make(map[string]bool)
	for name, ci := range c.Conditions {
		r, err := ci.IsTrueFor(o)
		if err != nil {
			log.Error(err)
		}
		if !r {
			result = false
		}
		condStates[name] = r
	}

	return result, condStates
}

//BuildCondition covers builds a user has/had running
type BuildCondition struct {
	Condition
	IdleAfter time.Duration
}

func NewBuildCondition(idleAfter time.Duration) *BuildCondition {
	b := &BuildCondition{IdleAfter: idleAfter}
	return b
}

//IsTrueFor returns true if a User does not have any Builds or does not have any
//Active builds and last Done build happened IdleAfter (time.Duration) before
func (c *BuildCondition) IsTrueFor(object interface{}) (result bool, err error) {
	result = false
	u, ok := object.(*model.User)
	if !ok {
		return false, errors.New(fmt.Sprintf("%s is not of type *User", object))
	}

	if !u.HasBuilds() || (!u.HasActive() && u.DoneBuild.Status.CompletionTimestamp.Time.Add(c.IdleAfter).Before(time.Now())) {
		result = true
	}

	return result, err

}

//UserCondition covers information about User passed from Proxy
type UserCondition struct {
	Condition
	IdleAfter time.Duration
	proxyURL  string
}

func NewUserCondition(proxyURL string, idleAfter time.Duration) *UserCondition {
	b := &UserCondition{
		proxyURL:  proxyURL,
		IdleAfter: idleAfter,
	}
	return b
}

//IsTrueFor returns true if there are no buffered request, last request was forwarded at least before
//IdleAfter (time.Duration) and the user accessed Jenkins UI at least before IdleAfter
func (c *UserCondition) IsTrueFor(object interface{}) (result bool, err error) {
	result = false
	b, ok := object.(*model.User)
	if !ok {
		return false, errors.New(fmt.Sprintf("%s is not of type *User", object))
	}

	url := fmt.Sprintf("%s/papi/info/%s", c.proxyURL, fmt.Sprintf("%s-jenkins", b.Name)) //FIX sprintf!
	resp, err := http.Get(url)
	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	if resp.StatusCode != 200 {
		log.Error(fmt.Sprintf("Got status %s fom %s", resp.Status, url))
		return result, errors.New(resp.Status)
	}

	ar := &proxyAPI.APIResponse{}
	err = json.Unmarshal(body, &ar)
	if err != nil {
		return result, err
	}
	if ar.Requests > 0 {
		return result, nil
	}

	tu := time.Unix(ar.LastVisit, 0)
	tr := time.Unix(ar.LastRequest, 0)
	if err != nil {
		return result, err
	}
	if tu.Add(c.IdleAfter).Before(time.Now()) && tr.Add(c.IdleAfter).Before(time.Now()) {
		result = true
	}

	return result, err

}

//DCCondition covers changes to Jenkins DeploymentConfigs
type DCCondition struct {
	Condition
	IdleAfter time.Duration
}

func NewDCCondition(idleAfter time.Duration) *DCCondition {
	b := &DCCondition{
		IdleAfter: idleAfter,
	}
	return b
}

//IsTrueFor returns true if the last change to DC happened at least
//before IdleAfter (time.Duration)
func (c *DCCondition) IsTrueFor(object interface{}) (result bool, err error) {
	result = false
	b, ok := object.(*model.User)
	if !ok {
		return false, errors.New(fmt.Sprintf("%s is not of type *User", object))
	}

	if b.JenkinsLastUpdate.Add(c.IdleAfter).Before(time.Now()) {
		result = true
	}

	return
}
