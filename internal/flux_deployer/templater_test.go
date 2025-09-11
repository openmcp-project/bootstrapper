package flux_deployer_test

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
)

// TestTemplateDirectory tests the TemplateDirectory function by templating all files in directory testdata/03/templates.
func TestTemplateDirectory(t *testing.T) {
	ti := flux_deployer.TemplateInput{
		"test1": "foo",
		"test2": "bar",
	}
	resultDir := t.TempDir()
	err := flux_deployer.TemplateDirectory("./testdata/03/templates", resultDir, ti, logging.GetLogger())
	assert.NoError(t, err, "error templating directory")

	contentA, err := os.ReadFile(path.Join(resultDir, "a.yaml"))
	assert.NoError(t, err, "error reading templated a.yaml")
	assert.Equal(t, "test: foo", string(contentA), "templated content of a.yaml does not match expected")

	contentB, err := os.ReadFile(path.Join(resultDir, "b/c.yaml"))
	assert.NoError(t, err, "error reading templated b/c.yaml")
	assert.Equal(t, "test: bar", string(contentB), "templated content of b/c.yaml does not match expected")
}
