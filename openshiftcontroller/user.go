package openshiftcontroller

import (
	"fmt"
	//"fmt"
	"time"
	"encoding/json"
	"strings"
)

type User struct {
	ActiveBuilds map[string]Build
	DoneBuilds map[string]Build
	Name string
	JenkinsStateList []JenkinsState
}

func (u *User) HasActive() bool {
	return len(u.ActiveBuilds) > 0
}

func (u *User) LastActive() Build {
	var lastB Build
	for _, b := range u.ActiveBuilds {
		if lastB.Status.StartTimestamp.Time.Before(b.Status.StartTimestamp.Time) {
			lastB = b
		}
	}

	return lastB
}

func (u *User) LastDone() Build {
	var lastB Build
	for _, b := range u.DoneBuilds {
		if lastB.Status.CompletionTimestamp.Time.Before(b.Status.CompletionTimestamp.Time) {
			lastB = b
		}
	}

	return lastB
}

func (u *User) LastBuild() Build {
	if u.HasActive() {
		return u.LastActive()
	} else {
		return u.LastDone()
	}
}

func (u *User) AddJenkinsState(running bool, time time.Time, message string) {
	u.JenkinsStateList = append(u.JenkinsStateList, JenkinsState{Running: running, Time: time, Message: message})
	fmt.Printf("Recorded states: %d", len(u.JenkinsStateList))
}

type JenkinsState struct {
	Running bool
	Time time.Time
	Message string
}

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
	Name string `json:"name"`
	Namespace string `json:"namespace"`
	Annotations struct {
		BuildNumber string `json:"openshift.io/build.number"`
		JenkinsNamespace string `json:"openshift.io/jenkins-namespace"`
		IdledAt string `json:"idling.alpha.openshift.io/idled-at"`
	} `json:"annotations"`
}

type Status struct {
	Phase string `json:"phase"`
	StartTimestamp BuildTime `json:"startTimestamp"`
	CompletionTimestamp BuildTime `json:"completionTimestamp"`
}

type BuildTime struct {
	time.Time
}

func NewUser(n string) (u *User) {
	u = &User{
		Name: n,
		ActiveBuilds: make(map[string]Build),
		DoneBuilds: make(map[string]Build),
		JenkinsStateList: []JenkinsState{JenkinsState{true, time.Now().UTC(), "init"}},
	}

	return u
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