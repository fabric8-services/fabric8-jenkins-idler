package mock

import (
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/toggles"
	"github.com/fabric8-services/fabric8-jenkins-idler/internal/util"
)

type featureToggle struct {
	uuids []string
}

func NewMockFeatureToggle(validIds []string) toggles.Features {
	return &featureToggle{uuids: validIds}
}

func (m *featureToggle) IsIdlerEnabled(uid string) (bool, error) {
	return util.Contains(m.uuids, uid), nil
}
