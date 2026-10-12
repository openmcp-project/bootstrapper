package ocm_cli_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	testutil "github.com/openmcp-project/bootstrapper/test/utils"
)

const (
	ocmCmdGet              = "get"
	ocmCmdComponentVersion = "componentversion"
	ocmArgOutput           = "--output"
	ocmArgOutputYAML       = "yaml"
)

func TestExecute(t *testing.T) {
	expectError := errors.New("expected error")

	testutil.DownloadOCMAndAddToPath(t)

	ctfIn := testutil.BuildComponent("./testdata/01/component-constructor.yaml", t)

	testCases := []struct {
		desc          string
		commands      []string
		arguments     []string
		ocmConfig     string
		expectedError error
	}{
		{
			desc:          "get componentversion",
			commands:      []string{ocmCmdGet, ocmCmdComponentVersion},
			arguments:     []string{ocmArgOutput, ocmArgOutputYAML, ctfIn},
			ocmConfig:     ocmcli.NoOcmConfig,
			expectedError: nil,
		},
		{
			desc:          "get componentversion with invalid argument",
			commands:      []string{ocmCmdGet, ocmCmdComponentVersion},
			arguments:     []string{ocmArgOutput, ocmArgOutputYAML, "invalid-argument"},
			ocmConfig:     ocmcli.NoOcmConfig,
			expectedError: expectError,
		},
		{
			desc:          "get componentversion with ocm config",
			commands:      []string{ocmCmdGet, ocmCmdComponentVersion},
			arguments:     []string{ocmArgOutput, ocmArgOutputYAML, ctfIn},
			ocmConfig:     "./testdata/01/ocm-config.yaml",
			expectedError: nil,
		},

		{
			desc:          "get componentversion with unsupported ocm config",
			commands:      []string{ocmCmdGet, ocmCmdComponentVersion},
			arguments:     []string{ocmArgOutput, ocmArgOutputYAML, ctfIn},
			ocmConfig:     "./testdata/01/unsupported-ocm-config.yaml",
			expectedError: expectError,
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

func TestFilterComponentVersionsByLabel(t *testing.T) {
	cvs := []ocmcli.ComponentVersion{
		{Component: ocmcli.Component{Name: "a", Version: "v1.0.0", Labels: []ocmcli.Label{{Name: "env", Value: "prod"}, {Name: "team", Value: "alpha"}}}},
		{Component: ocmcli.Component{Name: "b", Version: "v1.1.0", Labels: []ocmcli.Label{{Name: "env", Value: "prod"}}}},
		{Component: ocmcli.Component{Name: "c", Version: "v1.2.0", Labels: []ocmcli.Label{{Name: "env", Value: "staging"}}}},
		{Component: ocmcli.Component{Name: "d", Version: "v1.3.0", Labels: nil}},
	}

	testCases := []struct {
		desc          string
		filter        []ocmcli.Label
		requireLabel  bool
		expectedNames []string
	}{
		{
			desc:          "require single label match",
			filter:        []ocmcli.Label{{Name: "env", Value: "prod"}},
			requireLabel:  true,
			expectedNames: []string{"a", "b"},
		},
		{
			desc:          "require multiple label match",
			filter:        []ocmcli.Label{{Name: "env", Value: "prod"}, {Name: "team", Value: "alpha"}},
			requireLabel:  true,
			expectedNames: []string{"a"},
		},
		{
			desc:          "exclude single label match",
			filter:        []ocmcli.Label{{Name: "env", Value: "prod"}},
			requireLabel:  false,
			expectedNames: []string{"c", "d"},
		},
		{
			desc:          "require label not present in any cv",
			filter:        []ocmcli.Label{{Name: "env", Value: "canary"}},
			requireLabel:  true,
			expectedNames: []string{},
		},
		{
			desc:          "empty filter with require returns all",
			filter:        []ocmcli.Label{},
			requireLabel:  true,
			expectedNames: []string{"a", "b", "c", "d"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := ocmcli.FilterComponentVersionsByLabel(cvs, tc.filter, tc.requireLabel)
			names := make([]string, len(result))
			for i, cv := range result {
				names[i] = cv.Component.Name
			}
			assert.ElementsMatch(t, tc.expectedNames, names)
		})
	}
}

func TestGetComponentVersion(t *testing.T) {
	expectError := errors.New("expected error")
	testutil.DownloadOCMAndAddToPath(t)

	ctfIn := testutil.BuildComponent("./testdata/01/component-constructor.yaml", t)

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

				cvMarshaled, err := json.Marshal(cv)
				assert.NoError(t, err)
				assert.NotNil(t, cvMarshaled)

				var cvAsMap map[string]interface{}
				err = json.Unmarshal(cvMarshaled, &cvAsMap)
				assert.NoError(t, err)
				assert.NotNil(t, cvAsMap)
				assert.Contains(t, cvAsMap, "component")
				assert.Contains(t, cvAsMap["component"], "componentReferences")
				comp := cvAsMap["component"].(map[string]interface{})
				assert.Len(t, comp["componentReferences"], 2)
			},
		},
		{
			desc:          "get component version with ocm config",
			componentRef:  ctfIn,
			ocmConfig:     "./testdata/01/ocm-config.yaml",
			expectedError: nil,
		},
		{
			desc:          "get component version with unsupported ocm config",
			componentRef:  ctfIn,
			ocmConfig:     "./testdata/01/unsupported-ocm-config.yaml",
			expectedError: expectError,
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
