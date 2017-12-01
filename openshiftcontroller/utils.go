package openshiftcontroller

import (
	"fmt"
	"errors"

	ic "github.com/fabric8-services/fabric8-jenkins-idler/clients"
)

func roundP(f float64) int {
	return int(f + 0.5)
}

//SplitGroups takes a list of strings and splits them in N groups evenly.
//N is given by length second parameter
func SplitGroups(data []string, split []*[]string) []*[]string {
	n := len(split)

	for i:=0;i<n;i++ {
		div := float64(len(data))/float64(n)
		start := div*float64(i)
		end := div*float64(i+1)

		p := make([]string, len(data[roundP(start):roundP(end)]))
		split[i] = &p

		copy(*split[i], data[roundP(start):roundP(end)])
	}

	return split
}


//GetLastBuild compares 2 builds and returns the newer one. There are differences between
//active (StartTimestamp) and done (CompletionTimestamp) builds.
func GetLastBuild(b1 *ic.Build, b2 *ic.Build) (*ic.Build, error) {
	if b1 == nil {
		return b2, nil
	} else if b2 == nil {
		return b1, nil
	}

	b1a := IsActive(b1)
	b2a := IsActive(b2)
	if b1a != b2a {
		return b1, errors.New(fmt.Sprintf("Cannot compare Active and Done builds - %s vs. %s", b1.Status.Phase, b2.Status.Phase))
	}

	if b1a && b2a {
		if b1.Status.StartTimestamp.Time.Before(b2.Status.StartTimestamp.Time) {
			return b2, nil
		} else {
			return b1, nil
		}
	} else {
		if b1.Status.CompletionTimestamp.Time.Before(b2.Status.CompletionTimestamp.Time) {
			return b2, nil
		} else {
			return b1, nil
		}
	}
}

//IsActive returns true ifa build phase suggests a build is active.
//It returns false otherwise.
func IsActive(b *ic.Build) bool {
	return ic.Phases[b.Status.Phase] == 1
}