package clients

import (
	"fmt"
	"encoding/json"
	"time"
	"strings"
)

//OpenShift related structs

const (
	JenkinsIdled = 0
	JenkinsStarting = 1
	JenkinsRunning = 2
)

type Object struct {
	Type string `json:"type"`
	Object Build `json:"object"`
}

type DCObject struct {
	Type string `json:"type"`
	Object DeploymentConfig `json:"object"`
}

type BuildList struct {
	Kind string
	Items []Build `json:"items"`
}

type Build struct {
	Metadata Metadata `json:"metadata"`
	Status Status `json:"status"`
	Spec Spec `json:"spec"`
}

type Metadata struct {
	Name string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Annotations Annotations `json:"annotations"`
	Generation int
}

type Annotations struct {
	BuildNumber string `json:"openshift.io/build.number,omitempty"`
	JenkinsNamespace string `json:"openshift.io/jenkins-namespace,omitempty"`
	IdledAt string `json:"idling.alpha.openshift.io/idled-at,omitempty"`
	UnidleTargets string `json:"idling.alpha.openshift.io/unidle-targets,omitempty"`
	PrevScale string `json:"idling.alpha.openshift.io/previous-scale,omitempty"`
}

type Endpoint struct {
	Metadata Metadata `json:"metadata"`
}

type Status struct {
	Phase string `json:"phase"`
	StartTimestamp BuildTime `json:"startTimestamp"`
	CompletionTimestamp BuildTime `json:"completionTimestamp"`
}

type DeploymentConfig struct {
	Metadata Metadata `json:"metadata"`
	Status DCStatus `json:"status,omitempty"`
	Spec Spec `json:"spec,omitempty"`
}

type DCStatus struct {
	Replicas int
	ReadyReplicas int
	Conditions []Condition
	ObservedGeneration int `json:"observedGeneration,omitempty"`
	UnavailableReplicas int `json:"unavailableReplicas, omitempty"`
}

type Condition struct {
	Type string
	LastUpdateTime time.Time
	Status string
}

type Spec struct {
	Replicas int `json:"replicas"`
	Strategy Strategy
}

type Strategy struct {
	Type string
}

type Scale struct {
	Kind string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
	Metadata Metadata `json:"metadata"`
	Spec struct {
		Replicas int `json:"replicas"`
	} `json:"spec"`
}

type Projects struct {
	Items []*Project `json:"items"`
}

type Project struct {
	Metadata Metadata `json:"metadata"`
}

type BuildTime struct {
	time.Time
}

func (bt *BuildTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if len(s) == 0 {
		bt.Time = time.Now().UTC()
		return
	}
	bt.Time, err = time.Parse(time.RFC3339, s)

	return
}

func (s *Status) UnmarshalJSON(b []byte) (err error) {
	type LStatus Status
	ns := &LStatus{
		Phase: "",
		StartTimestamp: BuildTime{time.Now().UTC()},
		CompletionTimestamp: BuildTime{time.Now().UTC()},
	}

	if err := json.Unmarshal(b, ns); err != nil {
		return err
	}

	*s = Status(*ns)

	return
}

func (s DCStatus) GetByType(t string) (Condition, error) {
	for _, c := range s.Conditions {
		if c.Type == t {
			return c, nil
		}
	}

	return Condition{}, fmt.Errorf("Could not find condition '%s'", t)
}

var Phases = map[string]int {
	"Finished": 0,
	"Complete": 0,
	"Failed": 0,
	"Cancelled": 0,
	"Pending": 1,
	"New": 1,
	"Running": 1,
}
