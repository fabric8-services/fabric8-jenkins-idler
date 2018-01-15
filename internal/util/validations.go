package util

import (
	"fmt"
	"github.com/pkg/errors"
	"net/url"
	"strconv"
	"strings"
)

func IsNotEmpty(value interface{}, key string) error {
	s, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprintf("Value for %s needs to be a string.", key))
	}

	if len(s) == 0 {
		return errors.New(fmt.Sprintf("Value for %s cannot be empty.", key))
	}
	return nil

}

func IsURL(value interface{}, key string) error {
	s, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprintf("Value for %s needs to be a string.", key))
	}

	if !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "http://") {
		return errors.New(fmt.Sprintf("Value for %s needs to be a valid URL.", key))
	}

	_, err := url.ParseRequestURI(s)
	if err != nil {
		return errors.New(fmt.Sprintf("Value for %s needs to be a valid URL.", key))
	}
	return nil
}

func IsInt(value interface{}, key string) error {
	_, err := strconv.Atoi(value.(string))
	if err != nil {
		return errors.New(fmt.Sprintf("Value for %s needs to be an integer.", key))
	}
	return nil
}
