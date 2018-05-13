package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// OpenShift related structs

// PodState represents an different states of a Pod
type PodState int

const (
	// PodStateUnknown represents an unknown state of the Pod. Used usually with Error.
	PodStateUnknown PodState = 0
	// PodIdled represents the idled state of a Pod.
	PodIdled = 1
	// PodStarting state is when Pods are about to start.
	PodStarting = 2
	// PodRunning state is when Pods are running.
	PodRunning = 3
)

func (state PodState) String() string {
	states := [...]string{
		"unknown",
		"idled",
		"starting",
		"running",
	}
	if state < PodStateUnknown || state > PodRunning {
		state = 0
	}
	return states[state]
}

// Object is Build Object.
type Object struct {
	Type   string `json:"type"`
	Object Build  `json:"object"`
}

// DCObject is DeploymentConfig Object.
type DCObject struct {
	Type   string           `json:"type"`
	Object DeploymentConfig `json:"object"`
}

// BuildList is list of all Build.
type BuildList struct {
	Kind  string
	Items []Build `json:"items"`
}

// Build encapsulates the inputs needed to produce a new deployable image,
// as well as the status of the execution and a reference to the Pod which executed the build.
type Build struct {
	Metadata Metadata `json:"metadata"`
	Status   Status   `json:"status"`
	Spec     Spec     `json:"spec"`
}

// Metadata used in Build.
type Metadata struct {
	Name        string      `json:"name,omitempty"`
	Namespace   string      `json:"namespace,omitempty"`
	Annotations Annotations `json:"annotations"`
	Generation  int
}

// Annotations is a set of key, value pairs added to custom deployer and lifecycle pre/post hook pods.
// It contains imformation regarding build.
// It is used in Metadata as Annotations.
type Annotations struct {
	BuildNumber      string `json:"openshift.io/build.number,omitempty"`
	JenkinsNamespace string `json:"openshift.io/jenkins-namespace,omitempty"`
	IdledAt          string `json:"idling.alpha.openshift.io/idled-at,omitempty"`
	UnidleTargets    string `json:"idling.alpha.openshift.io/unidle-targets,omitempty"`
	PrevScale        string `json:"idling.alpha.openshift.io/previous-scale,omitempty"`
}

// Endpoint is the how a service is getting accessed.
// https://docs.openshift.com/online/rest_api/api/v1.Endpoints.html
type Endpoint struct {
	Metadata Metadata `json:"metadata"`
}

// Status is the current status of the build.
type Status struct {
	Phase               string    `json:"phase"`
	StartTimestamp      BuildTime `json:"startTimestamp"`
	CompletionTimestamp BuildTime `json:"completionTimestamp"`
}

// DeploymentConfig define the template for a pod and manages deploying new images or configuration changes.
// A single deployment configuration is usually analogous to a single micro-service.
type DeploymentConfig struct {
	Metadata Metadata `json:"metadata"`
	Status   DCStatus `json:"status,omitempty"`
	Spec     Spec     `json:"spec,omitempty"`
}

// DCStatus represents the current deployment state.
type DCStatus struct {
	Replicas            int
	ReadyReplicas       int
	Conditions          []Condition
	ObservedGeneration  int `json:"observedGeneration,omitempty"`
	UnavailableReplicas int `json:"unavailableReplicas,omitempty"`
}

// Condition covers changes to Build.
type Condition struct {
	Type           string
	LastUpdateTime time.Time
	Status         string
}

// Spec holds all the input necessary to produce a new build, and the conditions when to trigger them.
type Spec struct {
	Replicas int `json:"replicas"`
	Strategy Strategy
}

// Strategy defines how to perform a build.
// https://docs.openshift.com/online/dev_guide/builds/build_strategies.html
type Strategy struct {
	Type string
}

// Scale represents a scaling request for a resource.
type Scale struct {
	Kind       string   `json:"kind"`
	APIVersion string   `json:"apiVersion"`
	Metadata   Metadata `json:"metadata"`
	Spec       struct {
		Replicas int `json:"replicas"`
	} `json:"spec"`
}

// Projects is List of all Project.
type Projects struct {
	Items []*Project `json:"items"`
}

// Project is a unit of isolation and collaboration in OpenShift.
// https://docs.openshift.com/online/rest_api/oapi/v1.Project.html
type Project struct {
	Metadata Metadata `json:"metadata"`
}

// BuildTime is duration of the Build.
type BuildTime struct {
	Time time.Time
}

// UnmarshalJSON gets time from raw (in the form of []byte) JSON object.
func (bt *BuildTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if len(s) == 0 {
		bt.Time = time.Now().UTC()
		return
	}
	bt.Time, err = time.Parse(time.RFC3339, s)

	return
}

// UnmarshalJSON gets a Status Object from raw bytes.
func (s *Status) UnmarshalJSON(b []byte) (err error) {
	type LStatus Status
	ns := &LStatus{
		Phase:               "",
		StartTimestamp:      BuildTime{time.Now().UTC()},
		CompletionTimestamp: BuildTime{time.Now().UTC()},
	}

	if err := json.Unmarshal(b, ns); err != nil {
		return err
	}

	*s = Status(*ns)

	return
}

// GetByType gets condition by its type from Conditions of DCStatus.
func (s DCStatus) GetByType(t string) (Condition, error) {
	for _, c := range s.Conditions {
		if c.Type == t {
			return c, nil
		}
	}

	return Condition{}, fmt.Errorf("could not find condition '%s'", t)
}

// Phases are points in the build lifecycle.
var Phases = map[string]int{
	"Finished":  0,
	"Complete":  0,
	"Failed":    0,
	"Cancelled": 0,
	"Pending":   1,
	"New":       1,
	"Running":   1,
}
