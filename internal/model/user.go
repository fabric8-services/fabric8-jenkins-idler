package model

import (
	"fmt"
	"time"
)

// User represents a single user (user namespace) in the system. Mainly, it holds information
// about latest builds and changes to Jenkins DC for the user, which is then used in decision
// whether to (un)idle Jenkins.
type User struct {
	Name              string
	ID                string
	ActiveBuild       Build
	DoneBuild         Build
	JenkinsStateList  []JenkinsState
	JenkinsLastUpdate time.Time
}

// JenkinsState defines the state information of current Jenkins
// such as whether running or not, since how long has it been running, etc.
type JenkinsState struct {
	Running bool
	Time    time.Time
	Message string
}

// NewUser creates a new instance of a User given an id and name.
func NewUser(id string, name string) User {
	return User{
		ID:          id,
		Name:        name,
		ActiveBuild: Build{Status: Status{Phase: "New"}},
		DoneBuild:   Build{},
	}
}

// HasActiveBuilds checks if current user has any active builds.
// If so true is returned, otherwise false.
func (u *User) HasActiveBuilds() bool {
	return len(u.ActiveBuild.Metadata.Name) > 0
}

// HasCompletedBuilds checks if current User has any completed/done builds,
// If so true is returned, otherwise false.
func (u *User) HasCompletedBuilds() bool {
	return len(u.DoneBuild.Metadata.Name) > 0
}

// LastBuild returns last Jenkins Build of the current user.
func (u *User) LastBuild() Build {
	if u.HasActiveBuilds() {
		return u.ActiveBuild
	}

	return u.DoneBuild
}

// HasBuilds checks if current User has any active od completed Builds,
// if so it returns true else false.
func (u *User) HasBuilds() bool {
	return u.HasActiveBuilds() || u.HasCompletedBuilds()
}

func (u *User) String() string {
	return fmt.Sprintf("HasBuilds:%t HasActiveBuilds:%t JenkinsLastUpdate:%v", u.HasBuilds(), u.HasActiveBuilds(), u.JenkinsLastUpdate.Format(time.RFC822))
}
