package util

import (
	"fmt"
	"strings"
)

// MultiError defines a list of errors
type MultiError struct {
	Errors []error
}

// Empty checks if current instance of MultiError is empty
func (m *MultiError) Empty() bool {
	return len(m.Errors) == 0
}

// Collect appends an error to current instance of MultiError
func (m *MultiError) Collect(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

// ToError takes all errors from current instance of MultiError, bundles them in a single error and returns it
func (m MultiError) ToError() error {
	if len(m.Errors) == 0 {
		return nil
	}

	errStrings := []string{}
	for _, err := range m.Errors {
		errStrings = append(errStrings, err.Error())
	}
	return fmt.Errorf(strings.Join(errStrings, "\n"))
}
