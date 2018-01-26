package condition

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/model"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

var logger = log.WithFields(log.Fields{"component": "user-condition"})

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

func NewUserCondition(proxyURL string, idleAfter time.Duration) Condition {
	b := &UserCondition{
		proxyURL:  proxyURL,
		idleAfter: idleAfter,
	}
	return b
}

// Eval returns true if there are no buffered request, the last forwarded request occurred more than UserCondition.idleAfter
// minutes ago and the user accessed the Jenkins UI more than UserCondition.idleAfter minutes ago.
func (c *UserCondition) Eval(object interface{}) (bool, error) {
	b, ok := object.(model.User)
	if !ok {
		return false, errors.New(fmt.Sprintf("%s is not of type User", object))
	}

	proxyResponse, err := c.getProxyResponse(b.Name)
	if err != nil {
		return false, err
	}

	if proxyResponse.Requests > 0 {
		return false, nil
	}

	tu := time.Unix(proxyResponse.LastVisit, 0)
	tr := time.Unix(proxyResponse.LastRequest, 0)
	if tu.Add(c.idleAfter).Before(time.Now()) && tr.Add(c.idleAfter).Before(time.Now()) {
		return true, nil
	}

	return false, nil
}

func (c *UserCondition) getProxyResponse(userName string) (*ProxyResponse, error) {
	url := fmt.Sprintf("%s/papi/info/%s-jenkins", c.proxyURL, userName)
	logger.WithField("url", url).Info("Accessing Proxy API.")
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
