package toggles

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

type fixedUUIDToggle struct {
	uuids []string
}

// NewFixedUUIDToggle creates an instance of fixedUUIDToggle.
func NewFixedUUIDToggle(uuids []string) (Features, error) {
	return &fixedUUIDToggle{uuids: uuids}, nil
}

// IsIdlerEnabled checks if idler is enabled for current fixedUUIDToggle.
func (t *fixedUUIDToggle) IsIdlerEnabled(uuid string) (bool, error) {
	return util.Contains(t.uuids, uuid), nil
}
