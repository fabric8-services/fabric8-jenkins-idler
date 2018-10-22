package condition

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithFields(logrus.Fields{"component": "user-condition"})

// ProxyResponse represents response provided by the Jenkins Proxy for a particular user.
type ProxyResponse struct {
	Namespace   string `json:"namespace"`
	Requests    int    `json:"requests"`
	LastVisit   int64  `json:"last_visit"`
	LastRequest int64  `json:"last_request"`
}

// UserCondition covers information about User as provided by the Jenkins Proxy.
type UserCondition struct {
	idleAfter time.Duration
	proxyURL  string
}

// NewUserCondition creates a new instance of Condition given a proxyURL and idleAfter.
func NewUserCondition(proxyURL string, idleAfter time.Duration) Condition {
	b := &UserCondition{
		proxyURL:  proxyURL,
		idleAfter: idleAfter,
	}
	return b
}

// Eval returns true if there are no buffered request, the last forwarded request occurred more than UserCondition.idleAfter
// minutes ago and the user accessed the Jenkins UI more than UserCondition.idleAfter minutes ago.
func (c *UserCondition) Eval(object interface{}) (Action, error) {
	u, ok := object.(model.User)
	if !ok {
		return NoAction, fmt.Errorf("%T is not of type User", object)
	}

	log := logger.WithFields(logrus.Fields{
		"id":        u.ID,
		"name":      u.Name,
		"component": "user-condition",
	})

	proxyResponse, err := c.getProxyResponse(u.Name)
	if err != nil {
		log.WithField("action", "none").Errorf("proxy returned error: %s", err)
		return NoAction, err
	}

	if proxyResponse.Requests > 0 {
		log.WithField("action", "unidle").Infof(
			"proxy is still serving requests %d", proxyResponse.Requests)
		return UnIdle, nil
	}

	lv := time.Unix(proxyResponse.LastVisit, 0)
	visitIdleTime := lv.Add(c.idleAfter)

	lr := time.Unix(proxyResponse.LastRequest, 0)
	reqIdleTime := lr.Add(c.idleAfter)

	now := time.Now().UTC()

	log.WithField("check", "proxy:last-visit").Infof(
		"check if %v has gone past last visit %v - %v, last request %v - %v ",
		now, lv, visitIdleTime, lr, reqIdleTime)

	if now.After(visitIdleTime) && now.After(reqIdleTime) {
		log.WithField("action", "idle").Infof(
			"%v (%v) has elapsed after last visit: %v last request: %v",
			c.idleAfter, now, lv, lr)
		return Idle, nil
	}

	log.WithField("action", "idle").Infof(
		"%v (%v) has not elapsed after last visit: %v last request: %v",
		c.idleAfter, now, lv, lr)
	return UnIdle, nil
}

func (c *UserCondition) getProxyResponse(userName string) (*ProxyResponse, error) {
	url := fmt.Sprintf("%s/api/info/%s-jenkins", c.proxyURL, userName)
	logger.WithField("url", url).Debug("Accessing Proxy API.")
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		logger.Error(fmt.Sprintf("Got status %s fom %s", resp.Status, url))
		return nil, errors.New(resp.Status)
	}

	proxyResponse := ProxyResponse{}
	err = json.Unmarshal(body, &proxyResponse)
	if err != nil {
		return nil, err
	}
	return &proxyResponse, err
}
