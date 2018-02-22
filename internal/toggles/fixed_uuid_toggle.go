package toggles

import "github.com/minishift/minishift/pkg/util/strings"

type fixedUuidToggle struct {
	uuids []string
}

func NewFixedUuidToggle(uuids []string) (Features, error) {
	return &fixedUuidToggle{uuids: uuids}, nil
}

func (t *fixedUuidToggle) IsIdlerEnabled(uuid string) (bool, error) {
	return strings.Contains(t.uuids, uuid), nil
}
