package model

import (
	"fmt"
	"time"
)

// User represents a single user (user namespace) in the system. Mainly, it holds information
// about latest builds and changes to Jenkins DC for the user, which is then used in decision
// whether to (un)idle Jenkins
type User struct {
	ActiveBuild      Build
	DoneBuild        Build
	Name             string
	JenkinsStateList []JenkinsState
	FailedPulls      int
	UnIdleRetried    int
	// TODO Issue #92 Is this the right name for this field?
	JenkinsLastUpdate time.Time
	ID                string
}

func NewUser(id string, name string, isRunning bool) User {
	return User{
		ID:               id,
		Name:             name,
		ActiveBuild:      Build{Status: Status{Phase: "New"}},
		DoneBuild:        Build{},
		JenkinsStateList: []JenkinsState{{isRunning, time.Now().UTC(), "init"}},
		FailedPulls:      0,
		UnIdleRetried:    0,
	}
}

func (u *User) HasActiveBuilds() bool {
	return len(u.ActiveBuild.Metadata.Name) > 0
}

func (u *User) HasCompletedBuilds() bool {
	return len(u.DoneBuild.Metadata.Name) > 0
}

func (u *User) LastBuild() Build {
	if u.HasActiveBuilds() {
		return u.ActiveBuild
	} else {
		return u.DoneBuild
	}
}

func (u *User) HasBuilds() bool {
	return u.HasActiveBuilds() || u.HasCompletedBuilds()
}

func (u *User) AddJenkinsState(running bool, time time.Time, message string) {
	u.JenkinsStateList = append(u.JenkinsStateList, JenkinsState{Running: running, Time: time, Message: message})
}

func (u *User) String() string {
	return fmt.Sprintf("HasBuilds:%t HasActiveBuilds:%t JenkinsLastUpdate:%v", u.HasBuilds(), u.HasActiveBuilds(), u.JenkinsLastUpdate.Format(time.RFC822))
}

type JenkinsState struct {
	Running bool
	Time    time.Time
	Message string
}
