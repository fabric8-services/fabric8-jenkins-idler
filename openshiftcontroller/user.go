package openshiftcontroller

import (
	"time"

	"github.com/fabric8-services/fabric8-jenkins-idler/clients"
)

type User struct {
	ActiveBuild *clients.Build
	DoneBuild *clients.Build
	Name string
	JenkinsStateList []JenkinsState
	FailedPulls int
}

func (u *User) HasActive() bool {
	return u.ActiveBuild != nil
}

func (u *User) LastBuild() clients.Build {
	if u.HasActive() {
		return *u.ActiveBuild
	} else {
		return *u.DoneBuild
	}
}

func (u *User) HasBuilds() bool {
	return u.ActiveBuild != nil || u.DoneBuild != nil
}

func (u *User) AddJenkinsState(running bool, time time.Time, message string) {
	u.JenkinsStateList = append(u.JenkinsStateList, JenkinsState{Running: running, Time: time, Message: message})
}

type JenkinsState struct {
	Running bool
	Time time.Time
	Message string
}



func NewUser(n string, isRunning bool) (u *User) {
	u = &User{
		Name: n,
		ActiveBuild: nil,
		DoneBuild: nil,
		JenkinsStateList: []JenkinsState{JenkinsState{isRunning, time.Now().UTC(), "init"}},
		FailedPulls: 0,
	}

	return u
}