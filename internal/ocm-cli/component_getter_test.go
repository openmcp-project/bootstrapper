package ocm_cli_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	testutil "github.com/openmcp-project/bootstrapper/test/utils"
)

func TestComponentGetter(t *testing.T) {
	const (
		rootComponentName      = "github.com/openmcp-project/openmcp"
		templatesComponentName = "github.com/openmcp-project/gitops-templates"
	)

	testutil.DownloadOCMAndAddToPath(t)
	ctf := testutil.BuildComponent("./testdata/02/component-constructor.yaml", t)
	rootLocation := ctf + "//github.com/openmcp-project/openmcp:v0.0.11"
	g := ocmcli.NewComponentGetter(rootLocation, "gitops-templates/test-resource", ocmcli.NoOcmConfig)

	err := g.InitializeComponents(t.Context())
	assert.NoError(t, err, "Error initializing components")

	rootComponentVersion := g.RootComponentVersion()
	assert.NotNil(t, rootComponentVersion, "Root component version should not be nil")
	assert.Equal(t, rootComponentName, rootComponentVersion.Component.Name, "Root component name should match")

	templatesComponentVersion := g.TemplatesComponentVersion()
	assert.NotNil(t, templatesComponentVersion, "Templates component version should not be nil")
	assert.Equal(t, templatesComponentName, templatesComponentVersion.Component.Name, "Templates component name should match")
}
