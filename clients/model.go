package clients

import (
	"encoding/json"
	"time"
	"strings"
)

//OpenShift related structs

type Object struct {
	Type string `json:"type"`
	Object Build `json:"object"`
}

type BuildList struct {
	Kind string
	Items []Build `json:"items"`
}

type Build struct {
	Metadata Metadata `json:"metadata"`
	Status Status `json:"status"`
}

type Metadata struct {
	Name string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Annotations Annotations `json:"annotations"`
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
}

type Spec struct {
	Replicas int `json:"replicas"`
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

var Phases = map[string]int {
	"Finished": 0,
	"Complete": 0,
	"Failed": 0,
	"Cancelled": 0,
	"Pending": 1,
	"New": 1,
	"Running": 1,
}

var JenkinsStates = map[string]int {
	"Idled": 0,
	"Starting": 1,
	"Running": 2,
}