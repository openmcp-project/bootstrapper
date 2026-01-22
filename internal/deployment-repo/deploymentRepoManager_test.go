package deploymentrepo_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openmcp-project/bootstrapper/internal/config"

	deploymentrepo "github.com/openmcp-project/bootstrapper/internal/deployment-repo"
	testutils "github.com/openmcp-project/bootstrapper/test/utils"
)

const (
	incomingBranch = "incoming"
	outgoingBranch = "outgoing"
)

func TestDeploymentRepoManager(t *testing.T) {
	testutils.DownloadOCMAndAddToPath(t)

	// Component
	ctfIn := testutils.BuildComponent("./testdata/01/component-constructor.yaml", t)

	// Git repository
	originDir := t.TempDir()

	origin, err := git.PlainInit(originDir, false)
	assert.NoError(t, err)

	originWorkTree, err := origin.Worktree()
	assert.NoError(t, err)
	assert.NotNil(t, originWorkTree)

	dummyFilePath := filepath.Join(originDir, "dummy.txt")
	testutils.WriteToFile(t, dummyFilePath, "This is a dummy file.")
	testutils.AddFileToWorkTree(t, originWorkTree, "dummy.txt")
	testutils.WorkTreeCommit(t, originWorkTree, "Initial commit")

	// Configuration
	bootstrapConfig := &config.BootstrapperConfig{
		Component: config.Component{
			OpenMCPComponentLocation: ctfIn + "//github.com/openmcp-project/openmcp",
		},
		Environment: "dev",
		DeploymentRepository: config.DeploymentRepository{
			RepoURL:    originDir,
			PushBranch: incomingBranch,
			PullBranch: outgoingBranch,
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

	platformClient := fake.NewClientBuilder().Build()
	platformCluster := clusters.NewTestClusterFromClient("platform", platformClient)

	deploymentRepoManager := deploymentrepo.NewDeploymentRepoManager(
		bootstrapConfig,
		platformCluster,
		"./testdata/01/git-config.yaml",
		"",
		"./testdata/01/extra-manifests",
		"./testdata/01/patches/patches.yaml")
	assert.NotNil(t, deploymentRepoManager)

	_, err = deploymentRepoManager.Initialize(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.ApplyTemplates(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.ApplyProviders(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.ApplyCustomResourceDefinitions(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.ApplyExtraManifests(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.UpdateResourcesKustomization()
	assert.NoError(t, err)

	manifests, err := deploymentRepoManager.RunKustomize()
	assert.NoError(t, err)
	assert.NotEmpty(t, manifests)

	err = deploymentRepoManager.RunKustomizeAndApply(t.Context(), manifests)
	assert.NoError(t, err)

	commitMessage := "Apply deployment repo changes"
	err = deploymentRepoManager.CommitAndPushChanges(t.Context(), commitMessage)
	assert.NoError(t, err)

	// get the latest commit message to verify the push worked
	incomingBranchRef, err := origin.Reference(plumbing.NewBranchReferenceName(incomingBranch), true)
	assert.NoError(t, err)
	assert.Equal(t, "refs/heads/"+incomingBranch, incomingBranchRef.Name().String())
	originCommit, err := origin.CommitObject(incomingBranchRef.Hash())
	assert.NoError(t, err)
	assert.Equal(t, commitMessage, originCommit.Message)

	err = originWorkTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(incomingBranch),
	})
	assert.NoError(t, err)
	expectedRepoDir := "./testdata/01/expected-repo"
	actualRepoDir := originDir

	testutils.AssertDirectoriesEqualWithNormalization(t, expectedRepoDir, actualRepoDir, createTestNormalizer(originDir, ctfIn))

	fluxKustomization := &unstructured.Unstructured{}
	fluxKustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	})

	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "bootstrap", Namespace: "default"}, fluxKustomization)
	assert.NoError(t, err)
}

// createTestNormalizer returns a function that normalizes file content by replacing actual repository URLs with placeholders.
func createTestNormalizer(actualRepoURL, actualOCMRepoURL string) func(string, string) string {
	return func(content, filePath string) string {
		// For gitrepo.yaml files, replace the actual repo URL with a placeholder
		if strings.Contains(filePath, "gitrepo.yaml") {
			content = strings.ReplaceAll(content, actualRepoURL, "{{GIT_REPO_URL}}")
		}
		// For files that may contain OCM repository URLs, replace the actual repo URL with a placeholder
		if strings.Contains(filePath, ".yaml") || strings.Contains(filePath, ".yml") || strings.Contains(filePath, ".json") {
			content = strings.ReplaceAll(content, actualOCMRepoURL, "{{OCM_REPO_URL}}")
		}
		return content
	}
}
