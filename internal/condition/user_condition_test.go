package condition

import (
	"io/ioutil"
	"testing"

	"github.com/fabric8-services/fabric8-jenkins-idler/internal/testutils/common"
	"github.com/stretchr/testify/assert"
)

func Test_get_proxy_response(t *testing.T) {
	// TODO(chmouel): Need to get the actual response from the proxy, built this
	// json from reading proxy code.
	tenantData, err := ioutil.ReadFile("../testutils/testdata/proxy.json")
	if err != nil {
		assert.NoError(t, err)
	}
	srv := common.MockServer(tenantData)
	defer srv.Close()

	uc := UserCondition{
		proxyURL: srv.URL,
	}

	response, err := uc.getProxyResponse("test")
	if err != nil {
		assert.NoError(t, err)
	}
	assert.Equal(t, "test", response.Namespace)
}
