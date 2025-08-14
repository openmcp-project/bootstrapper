package flux_deployer_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
)

func LoadComponentVersion(t *testing.T, path string) *ocmcli.ComponentVersion {
	cv := &ocmcli.ComponentVersion{}
	content, err := os.ReadFile(path)
	assert.NoError(t, err, "error reading component version file")
	err = yaml.Unmarshal(content, cv)
	assert.NoError(t, err, "error unmarshalling component version")
	return cv
}
