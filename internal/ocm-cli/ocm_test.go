package ocm_cli_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	testutil "github.com/openmcp-project/bootstrapper/test/utils"
)

func TestExecute(t *testing.T) {
	expectError := errors.New("expected error")

	testutil.DownloadOCMAndAddToPath(t)

	ctfIn := testutil.BuildComponent("./testdata/component-constructor.yaml", t)

	testCases := []struct {
		desc          string
		commands      []string
		arguments     []string
		ocmConfig     string
		expectedError error
	}{
		{
			desc:          "get componentversion",
			commands:      []string{"get", "componentversion"},
			arguments:     []string{"--output", "yaml", ctfIn},
			ocmConfig:     ocmcli.NoOcmConfig,
			expectedError: nil,
		},
		{
			desc:          "get componentversion with invalid argument",
			commands:      []string{"get", "componentversion"},
			arguments:     []string{"--output", "yaml", "invalid-argument"},
			ocmConfig:     ocmcli.NoOcmConfig,
			expectedError: expectError,
		},
		{
			desc:          "get componentversion with ocm config",
			commands:      []string{"get", "componentversion"},
			arguments:     []string{"--output", "yaml", ctfIn},
			ocmConfig:     "./testdata/ocm-config.yaml",
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := ocmcli.Execute(t.Context(), tc.commands, tc.arguments, tc.ocmConfig)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetComponentVersion(t *testing.T) {
	expectError := errors.New("expected error")
	testutil.DownloadOCMAndAddToPath(t)

	ctfIn := testutil.BuildComponent("./testdata/component-constructor.yaml", t)

	testCases := []struct {
		desc          string
		componentRef  string
		ocmConfig     string
		expectedError error
		verify        func(cv *ocmcli.ComponentVersion)
	}{
		{
			desc:          "get component version",
			componentRef:  ctfIn,
			ocmConfig:     ocmcli.NoOcmConfig,
			expectedError: nil,
			verify: func(cv *ocmcli.ComponentVersion) {
				assert.Equal(t, cv.Component.Name, "github.com/openmcp-project/bootstrapper/test")
				assert.Equal(t, cv.Component.Version, "v0.0.1")
				assert.Len(t, cv.Component.ComponentReferences, 2)
				assert.Len(t, cv.Component.Resources, 1)

				assert.Contains(t, cv.Component.ComponentReferences, ocmcli.ComponentReference{
					Name:          "bootstrapper-dependency-a",
					Version:       "v0.2.0",
					ComponentName: "github.com/openmcp-project/bootstrapper-dependency-a",
				})
				assert.Contains(t, cv.Component.ComponentReferences, ocmcli.ComponentReference{
					Name:          "bootstrapper-dependency-b",
					Version:       "v0.3.0",
					ComponentName: "github.com/openmcp-project/bootstrapper-dependency-b",
				})

				assert.Contains(t, cv.Component.Resources, ocmcli.Resource{
					Name:    "test-resource",
					Version: "v0.0.1",
					Type:    "blob",
					Access: ocmcli.Access{
						Type:           "localBlob",
						LocalReference: cv.Component.Resources[0].Access.LocalReference,
						MediaType:      ptr.To("application/octet-stream"),
					},
				})
			},
		},
		{
			desc:          "get component version with ocm config",
			componentRef:  ctfIn,
			ocmConfig:     "./testdata/ocm-config.yaml",
			expectedError: nil,
		},
		{
			desc:          "get component version with invalid reference",
			componentRef:  "invalid-component-ref",
			ocmConfig:     ocmcli.NoOcmConfig,
			expectedError: expectError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cv, err := ocmcli.GetComponentVersion(t.Context(), tc.componentRef, tc.ocmConfig)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tc.verify != nil {
				tc.verify(cv)
			}
		})
	}
}
