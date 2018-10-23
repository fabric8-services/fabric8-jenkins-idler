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
	JenkinsLastUpdate time.Time
	IdleStatus        IdleStatus
}

// IdleStatus contains information about the idle/un-idle status like timestamp
// and reasons of failure
type IdleStatus struct {
	Timestamp time.Time
	Success   bool
	Reason    string
}

// NewIdleStatus returns IdleStatus based on the error provided
func NewIdleStatus(err error) IdleStatus {
	if err != nil {
		return IdleStatus{
			Timestamp: time.Now().UTC(),
			Success:   false,
			Reason:    fmt.Sprintf("Failed to idle with error: %v", err),
		}
	}
	return IdleStatus{
		Timestamp: time.Now().UTC(),
		Success:   true,
		Reason:    "Successfully idled",
	}
}

// NewUnidleStatus returns IdleStatus based on the error provided
func NewUnidleStatus(err error) IdleStatus {
	if err != nil {
		return IdleStatus{
			Timestamp: time.Now().UTC(),
			Success:   false,
			Reason:    fmt.Sprintf("Failed to un-idle with error: %v", err),
		}
	}
	return IdleStatus{
		Timestamp: time.Now().UTC(),
		Success:   true,
		Reason:    "Successfully un-idled",
	}
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

// StateDump returns a String representing the internal states like
// HasActiveBuilds, LastUpdate etc useful for debugging
func (u *User) StateDump() string {
	return fmt.Sprintf("HasBuilds:%t HasActiveBuilds:%t JenkinsLastUpdate:%v",
		u.HasBuilds(), u.HasActiveBuilds(), u.JenkinsLastUpdate.Format(time.RFC822))
}
