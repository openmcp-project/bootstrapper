package ocm_cli_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	testutil "github.com/openmcp-project/bootstrapper/test/utils"
)

func TestComponentGetter(t *testing.T) {
	const (
		componentA             = "github.com/openmcp-project/bootstrapper/test-component-getter-a"
		componentB             = "github.com/openmcp-project/bootstrapper/test-component-getter-b"
		componentC             = "github.com/openmcp-project/bootstrapper/test-component-getter-c"
		version001             = "v0.0.1"
		templatesComponentName = "github.com/openmcp-project/gitops-templates"
	)

	testutil.DownloadOCMAndAddToPath(t)
	ctf := testutil.BuildComponent("./testdata/02/component-constructor.yaml", t)

	testCases := []struct {
		desc                           string
		rootLocation                   string
		deployTemplates                string
		expectInitializationError      bool
		expectedTemplatesComponentName string
		expectResourceError            bool
		expectedResourceName           string
	}{
		{
			desc:                           "should get a resource of the root component",
			rootLocation:                   fmt.Sprintf("%s//%s:%s", ctf, componentA, version001),
			deployTemplates:                "test-resource-a",
			expectedTemplatesComponentName: componentA,
			expectedResourceName:           "test-resource-a",
		},
		{
			desc:                           "should get a resource of a referenced component",
			rootLocation:                   fmt.Sprintf("%s//%s:%s", ctf, componentA, version001),
			deployTemplates:                "reference-b/test-resource-b",
			expectedTemplatesComponentName: componentB,
			expectedResourceName:           "test-resource-b",
		},
		{
			desc:                           "should get a resource of a nested referenced component",
			rootLocation:                   fmt.Sprintf("%s//%s:%s", ctf, componentA, version001),
			deployTemplates:                "reference-b/reference-c/test-resource-c",
			expectedTemplatesComponentName: componentC,
			expectedResourceName:           "test-resource-c",
		},
		{
			desc:                      "should fail for an unknown root component",
			rootLocation:              fmt.Sprintf("%s//%s:%s", ctf, "unknown-component", version001),
			deployTemplates:           "reference-b/test-resource-b",
			expectInitializationError: true,
		},
		{
			desc:                      "should fail for an unknown component reference",
			rootLocation:              fmt.Sprintf("%s//%s:%s", ctf, componentA, version001),
			deployTemplates:           "unknown-reference/test-resource-b",
			expectInitializationError: true,
		},
		{
			desc:                           "should fail for an unknown resource",
			rootLocation:                   fmt.Sprintf("%s//%s:%s", ctf, componentA, version001),
			deployTemplates:                "reference-b/unknown-resource",
			expectedTemplatesComponentName: componentB,
			expectResourceError:            true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			g := ocmcli.NewComponentGetter(tc.rootLocation, tc.deployTemplates, ocmcli.NoOcmConfig)

			err := g.InitializeComponents(t.Context())
			if tc.expectInitializationError {
				assert.Error(t, err, "Expected an error initializing components")
				return
			}
			assert.NoError(t, err, "Error initializing components")

			rootComponentVersion := g.RootComponentVersion()
			assert.NotNil(t, rootComponentVersion, "Root component version should not be nil")
			assert.Equal(t, componentA, rootComponentVersion.Component.Name, "Root component name should match")

			templatesComponentVersion := g.TemplatesComponentVersion()
			assert.NotNil(t, templatesComponentVersion, "Templates component version should not be nil")
			assert.Equal(t, tc.expectedTemplatesComponentName, templatesComponentVersion.Component.Name, "Templates component name should match")

			resource, err := templatesComponentVersion.GetResource(g.TemplatesResourceName())
			if tc.expectResourceError {
				assert.Error(t, err, "Expected an error getting resource")
				return
			}
			assert.NoError(t, err, "Error getting resource")
			assert.Equal(t, tc.expectedResourceName, resource.Name, "Resource name should match")
		})
	}
}
