package openshiftcontroller

import (
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/clients"
)

//User represents a single user (user namespace) in the system. Mainly, it holds information
//about latest builds and changes to Jenkins DC for the user, which is then used in decision
//whether to (un)idle Jenkins
type User struct {
	ActiveBuild       *clients.Build
	DoneBuild         *clients.Build
	Name              string
	JenkinsStateList  []JenkinsState
	FailedPulls       int
	UnidleRetried     int
	JenkinsLastUpdate time.Time
	ID                string
}

func (u *User) HasActive() bool {
	return len(u.ActiveBuild.Metadata.Name) > 0
}

func (u *User) HasDone() bool {
	return len(u.DoneBuild.Metadata.Name) > 0
}

func (u *User) LastBuild() clients.Build {
	if u.HasActive() {
		return *u.ActiveBuild
	} else {
		return *u.DoneBuild
	}
}

func (u *User) HasBuilds() bool {
	return u.HasActive() || u.HasDone()
}

func (u *User) AddJenkinsState(running bool, time time.Time, message string) {
	u.JenkinsStateList = append(u.JenkinsStateList, JenkinsState{Running: running, Time: time, Message: message})
}

type JenkinsState struct {
	Running bool
	Time    time.Time
	Message string
}

func NewUser(id string, n string, isRunning bool) (u *User) {
	u = &User{
		ID:               id,
		Name:             n,
		ActiveBuild:      &clients.Build{Status: clients.Status{Phase: "New"}},
		DoneBuild:        &clients.Build{},
		JenkinsStateList: []JenkinsState{{isRunning, time.Now().UTC(), "init"}},
		FailedPulls:      0,
		UnidleRetried:    0,
	}

	return u
}
