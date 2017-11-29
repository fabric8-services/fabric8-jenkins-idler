package openshiftcontroller

import (
	"fmt"
	"time"
	"errors"
	"net/http"
	"io/ioutil"
	"encoding/json"

	proxyAPI "github.com/fabric8-services/fabric8-jenkins-proxy/api"

	log "github.com/sirupsen/logrus"
)

type ConditionI interface {
	IsTrueFor(object interface{}) (bool, error)
}

type Condition struct {
	ConditionI
}

type Conditions struct {
	Conditions map[string]ConditionI
}

func (c *Conditions) Eval(o interface{}) (result bool) {
	result = true
	for _, ci := range c.Conditions {
		r, err := ci.IsTrueFor(o)
		if err != nil {
			log.Error(err)
		} else if !r {
			result = false
		}
	}

	return result
}

type BuildCondition struct {
	Condition
	IdleAfter time.Duration
}

func NewBuildCondition(idleAfter time.Duration) *BuildCondition {
	b := &BuildCondition{IdleAfter: idleAfter}
	return b
}

func (c *BuildCondition) IsTrueFor(object interface{}) (result bool, err error) {
	result = false
	u, ok := object.(*User)
	if !ok {
		return false, errors.New(fmt.Sprintf("%s is not of type *User", object))
	}

	if !u.HasBuilds() || (!u.HasActive() && u.DoneBuild.Status.CompletionTimestamp.Time.Add(c.IdleAfter).Before(time.Now())) {
		result = true
	}

	return result, err

}

type UserCondition struct {
	Condition
	IdleAfter time.Duration
	proxyURL string
}

func NewUserCondition(proxyURL string, idleAfter time.Duration) *UserCondition {
	b := &UserCondition{
		proxyURL: proxyURL,
		IdleAfter: idleAfter,
	}
	return b
}

func (c *UserCondition) IsTrueFor(object interface{}) (result bool, err error) {
	result = false
	b, ok := object.(*User)
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

func (c *DCCondition) IsTrueFor(object interface{}) (result bool, err error) {
	result = false
	b, ok := object.(*User)
	if !ok {
		return false, errors.New(fmt.Sprintf("%s is not of type *User", object))
	}

	if b.JenkinsLastUpdate.Add(c.IdleAfter).Before(time.Now()) {
		result = true
	}

	return
}