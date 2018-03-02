package util

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// IsNotEmpty checks if value associated with the current key is not empty.
func IsNotEmpty(value interface{}, key string) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("value for %s needs to be a string", key)
	}

	if len(s) == 0 {
		return fmt.Errorf("value for %s cannot be empty", key)
	}
	return nil

}

// IsURL checks if value associated with the current key is a URL.
func IsURL(value interface{}, key string) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("value for %s needs to be a string", key)
	}

	if !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "http://") {
		return fmt.Errorf("value for %s needs to be a valid URL", key)
	}

	_, err := url.ParseRequestURI(s)
	if err != nil {
		return fmt.Errorf("value for %s needs to be a valid URL", key)
	}
	return nil
}

// IsInt checks if value associated with the current key is an int.
func IsInt(value interface{}, key string) error {
	_, err := strconv.Atoi(value.(string))
	if err != nil {
		return fmt.Errorf("value for %s needs to be an integer", key)
	}
	return nil
}

// IsBool checks if value associated with the current key is a bool.
func IsBool(value interface{}, key string) error {
	_, err := strconv.ParseBool(value.(string))
	if err != nil {
		return fmt.Errorf("value for %s needs to be an bool", key)
	}
	return nil
}
