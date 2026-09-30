package ocm_cli_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/json"
	descriptor "ocm.software/open-component-model/bindings/go/descriptor/v2"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"

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

func TestGetComponentVersion(t *testing.T) {
	expectError := errors.New("expected error")
	testutil.DownloadOCMAndAddToPath(t)

	ctfIn := testutil.BuildComponent("./testdata/01/component-constructor.yaml", t)
	cvRef := ctfIn + "//github.com/openmcp-project/bootstrapper/test:v0.0.1"

	testCases := []struct {
		desc          string
		componentRef  string
		ocmConfig     string
		expectedError error
		verify        func(cv *ocmcli.ComponentVersion)
	}{
		{
			desc:          "get component version",
			componentRef:  cvRef,
			ocmConfig:     ocmcli.NoOcmConfig,
			expectedError: nil,
			verify: func(cv *ocmcli.ComponentVersion) {
				assert.Equal(t, cv.Component.Name, "github.com/openmcp-project/bootstrapper/test")
				assert.Equal(t, cv.Component.Version, "v0.0.1")
				assert.Len(t, cv.Component.References, 2)
				require.Len(t, cv.Component.Resources, 1)

				type ref struct{ name, component, version string }
				refs := make([]ref, 0, len(cv.Component.References))
				for _, r := range cv.Component.References {
					refs = append(refs, ref{r.Name, r.Component, r.Version})
				}
				assert.Contains(t, refs, ref{"bootstrapper-dependency-a", "github.com/openmcp-project/bootstrapper-dependency-a", "v0.2.0"})
				assert.Contains(t, refs, ref{"bootstrapper-dependency-b", "github.com/openmcp-project/bootstrapper-dependency-b", "v0.3.0"})

				res := cv.Component.Resources[0]
				assert.Equal(t, "test-resource", res.Name)
				assert.Equal(t, "v0.0.1", res.Version)
				assert.Equal(t, "blob", res.Type)
				require.NotNil(t, res.Access)
				assert.Equal(t, "LocalBlob/v1", res.Access.Type.String())
				_, err := ocmcli.ImageReference(&res)
				assert.Error(t, err, "a local blob has no OCI image reference")

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
			componentRef:  cvRef,
			ocmConfig:     "./testdata/01/ocm-config.yaml",
			expectedError: nil,
		},
		{
			desc:          "get component version with unsupported ocm config",
			componentRef:  cvRef,
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

func TestListComponentVersions(t *testing.T) {
	testutil.DownloadOCMAndAddToPath(t)

	ctf := testutil.BuildComponent("../deployment-repo/testdata/01/component-constructor.yaml", t)

	cv := ocmcli.ComponentVersion{Repository: ctf}
	cv.Component.Name = "github.com/openmcp-project/openmcp/releasechannel/crossplane"

	versions, err := cv.ListComponentVersions(t.Context(), ocmcli.NoOcmConfig)
	assert.NoError(t, err)
	assert.Equal(t, []string{"v0.0.1", "v0.0.2"}, versions)
}

func TestImageReference(t *testing.T) {
	testCases := []struct {
		desc        string
		access      string
		expected    string
		expectError bool
	}{
		{
			desc:     "legacy ociArtifact access",
			access:   `{"type":"ociArtifact","imageReference":"ghcr.io/a/b:v1"}`,
			expected: "ghcr.io/a/b:v1",
		},
		{
			desc:     "versioned OCIImage access",
			access:   `{"type":"OCIImage/v1","imageReference":"ghcr.io/a/b:v1"}`,
			expected: "ghcr.io/a/b:v1",
		},
		{
			desc:        "local blob access",
			access:      `{"type":"LocalBlob/v1","localReference":"sha256:abc","mediaType":"application/x-tar"}`,
			expectError: true,
		},
		{
			desc:        "no access",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			res := descriptor.Resource{}
			res.Name = "r"
			if tc.access != "" {
				res.Access = &ocmruntime.Raw{}
				require.NoError(t, json.Unmarshal([]byte(tc.access), res.Access))
			}

			imageRef, err := ocmcli.ImageReference(&res)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, imageRef)
		})
	}
}
