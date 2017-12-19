package openshiftcontroller

import (
	"errors"
	"sort"
	"strings"
)

func FilterProjects(projects []string, filter []string) []string {
	result := make([]string, 0)
	for _, proj := range projects {
		isAllowed := false
		for _, f := range filter {
			if strings.HasPrefix(proj, f) {
				isAllowed = true
			}
		}
		if len(filter) == 0 || isAllowed {
			result = append(result, proj)
		}
	}

	return result
}

func AppendProject(groups []*[]string, project string) ([]*[]string, error) {
	s := 0
	if len(groups) == 0 {
		return groups, errors.New("Cannot append project to empty groups.")
	}

	if len(project) == 0 {
		return groups, nil
	}
	l := len(*groups[0])
	for i, g := range groups {
		if len(*g) < l {
			l = len(*g)
			s = i
		}
	}

	g := append(*groups[s], project)
	groups[s] = &g

	return groups, nil
}

func UpdateProjects(groups []*[]string, projects []string) ([]*[]string, error) {
	if len(groups) == 0 {
		return groups, errors.New("Cannot append project to empty groups.")
	}

	if len(projects) == 0 {
		return groups, nil
	}

	for _, g := range groups {
		for _, p := range *g {
			e := sort.SearchStrings(projects, p)
			if e < len(projects) && projects[e] == p {
				projects = append(projects[:e], projects[e+1:]...)
			}
		}
	}

	for _, p := range projects {
		groups, err := AppendProject(groups, p)
		if err != nil {
			return groups, err
		}
	}
	return groups, nil
}
