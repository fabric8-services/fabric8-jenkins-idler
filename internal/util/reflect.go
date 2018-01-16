package util

import (
	"runtime"
	"strings"
)

func NameOfFunction(pc uintptr) string {
	name := ""
	if rf := runtime.FuncForPC(pc); rf != nil {
		split := strings.Split(rf.Name(), ".")
		name = split[len(split)-1]
	}

	return name
}
