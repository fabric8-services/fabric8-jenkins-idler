package toggles

import "github.com/minishift/minishift/pkg/util/strings"

type fixedUUIDToggle struct {
	uuids []string
}

// NewFixedUUIDToggle creates an instance of fixedUUIDToggle.
func NewFixedUUIDToggle(uuids []string) (Features, error) {
	return &fixedUUIDToggle{uuids: uuids}, nil
}

// IsIdlerEnabled checks if idler is enabled for current fixedUUIDToggle.
func (t *fixedUUIDToggle) IsIdlerEnabled(uuid string) (bool, error) {
	return strings.Contains(t.uuids, uuid), nil
}
