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
			RepoBranch: "incoming",
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
		"./testdata/01/extra-manifests")
	assert.NotNil(t, deploymentRepoManager)

	_, err = deploymentRepoManager.Initialize(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.ApplyTemplates(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.ApplyProviders(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.ApplyCustomResourceDefinitions(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.UpdateResourcesKustomization()
	assert.NoError(t, err)

	err = deploymentRepoManager.RunKustomizeAndApply(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.CommitAndPushChanges(t.Context())
	assert.NoError(t, err)

	err = originWorkTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("incoming"),
	})
	assert.NoError(t, err)
	expectedRepoDir := "./testdata/01/expected-repo"
	actualRepoDir := originDir

	testutils.AssertDirectoriesEqualWithNormalization(t, expectedRepoDir, actualRepoDir, createGitRepoNormalizer(originDir))

	clusterProviderTest := &unstructured.Unstructured{}
	clusterProviderTest.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "openmcp.cloud",
		Version: "v1alpha1",
		Kind:    "ClusterProvider",
	})

	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "test"}, clusterProviderTest)
	assert.NoError(t, err)
	assert.Equal(t, "test", clusterProviderTest.GetName())

	serviceProviderTest := &unstructured.Unstructured{}
	serviceProviderTest.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "openmcp.cloud",
		Version: "v1alpha1",
		Kind:    "ServiceProvider",
	})

	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "test"}, serviceProviderTest)
	assert.NoError(t, err)
	assert.Equal(t, "test", serviceProviderTest.GetName())

	platformServiceTest := &unstructured.Unstructured{}
	platformServiceTest.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "openmcp.cloud",
		Version: "v1alpha1",
		Kind:    "PlatformService",
	})

	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "test"}, platformServiceTest)
	assert.NoError(t, err)
	assert.Equal(t, "test", platformServiceTest.GetName())
}

// CreateGitRepoNormalizer creates a normalizer function that replaces dynamic git repository URLs
func createGitRepoNormalizer(actualRepoURL string) func(string, string) string {
	return func(content, filePath string) string {
		// For gitrepo.yaml files, replace the actual repo URL with a placeholder
		if strings.Contains(filePath, "gitrepo.yaml") {
			return strings.ReplaceAll(content, actualRepoURL, "{{GIT_REPO_URL}}")
		}
		return content
	}
}
