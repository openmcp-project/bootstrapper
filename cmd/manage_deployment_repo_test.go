package cmd_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/openmcp-project/bootstrapper/cmd"
	"github.com/openmcp-project/bootstrapper/internal/config"
	testutil "github.com/openmcp-project/bootstrapper/test/utils"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestManageDeploymentRepo(t *testing.T) {
	//expectError := errors.New("expected error")

	testutil.DownloadOCMAndAddToPath(t)

	ctfIn := testutil.BuildComponent("../internal/deployment-repo/testdata/01/component-constructor.yaml", t)

	// Git repository
	originDir := t.TempDir()

	origin, err := git.PlainInit(originDir, false)
	assert.NoError(t, err)

	originWorkTree, err := origin.Worktree()
	assert.NoError(t, err)
	assert.NotNil(t, originWorkTree)

	dummyFilePath := filepath.Join(originDir, "dummy.txt")
	testutil.WriteToFile(t, dummyFilePath, "This is a dummy file.")
	testutil.AddFileToWorkTree(t, originWorkTree, "dummy.txt")
	testutil.WorkTreeCommit(t, originWorkTree, "Initial commit")

	// Configuration
	bootstrapConfig := &config.BootstrapperConfig{
		Component: config.Component{
			OpenMCPComponentLocation: ctfIn + "//github.com/openmcp-project/openmcp",
		},
		Environment: "dev",
		DeploymentRepository: config.DeploymentRepository{
			RepoURL:    originDir,
			PushBranch: "incoming",
			PullBranch: "outgoing",
		},
		OpenMCPOperator: config.OpenMCPOperator{
			Config: json.RawMessage(`{"someKey": "someValue"}`),
		},
		Providers: config.Providers{
			ClusterProviders: []config.Provider{
				{
					Name:   "test",
					Config: json.RawMessage(`{"verbosity": "info"}`),
				},
			},
			ServiceProviders: []config.Provider{
				{
					Name: "test",
				},
			},
			PlatformServices: []config.Provider{
				{
					Name: "test",
				},
			},
		},
		ImagePullSecrets: []string{"imgpull-a", "imgpull-b"},
		TemplateInput: map[string]interface{}{
			"additionalKey1": "additionalValue1",
			"additionalKey2": "additionalValue2",
			"myMap": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			"funkyVar": "funkyValue",
		},
	}

	bootstrapConfig.SetDefaults()
	err = bootstrapConfig.Validate()
	assert.NoError(t, err)

	bootstrapConfigMarshaled, err := yaml.Marshal(bootstrapConfig)
	assert.NoError(t, err)

	configFilePath := filepath.Join(t.TempDir(), "config.yaml")
	testutil.WriteToFile(t, configFilePath, string(bootstrapConfigMarshaled))

	testCases := []struct {
		desc          string
		arguments     []string
		flags         map[string]string
		expectedError error
	}{
		{
			desc: "dry-run",
			arguments: []string{
				configFilePath,
			},
			flags: map[string]string{
				cmd.FlagDryRun:               "",
				cmd.FlagPrintKustomized:      "",
				cmd.FlagGitConfig:            "../internal/deployment-repo/testdata/01/git-config.yaml",
				cmd.FlagExtraManifestDir:     "../internal/deployment-repo/testdata/01/extra-manifests",
				cmd.FlagKustomizationPatches: "../internal/deployment-repo/testdata/01/patches/patches.yaml",
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			root := cmd.RootCmd
			args := []string{"manage-deployment-repo"}
			if len(tc.arguments) > 0 {
				args = append(args, tc.arguments...)
			}

			// Add flags to args
			for flag, value := range tc.flags {
				if value == "" {
					args = append(args, "--"+flag)
				} else {
					args = append(args, "--"+flag, value)
				}
			}

			root.SetArgs(args)

			err := root.Execute()
			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
