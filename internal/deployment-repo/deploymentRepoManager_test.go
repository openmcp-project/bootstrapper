package deploymentrepo_test

import (
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	kimage "sigs.k8s.io/kustomize/pkg/image"
	"sigs.k8s.io/yaml"

	deploymentrepo "github.com/openmcp-project/bootstrapper/internal/deployment-repo"
	testutils "github.com/openmcp-project/bootstrapper/test/utils"
)

func TestDeploymentRepoManager(t *testing.T) {
	testutils.DownloadOCMAndAddToPath(t)

	ctfIn := testutils.BuildComponent("./testdata/01/component-constructor.yaml", t)

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

	clusterProviders := []string{"test"}
	serviceProviders := []string{"test"}
	platformServices := []string{"test"}
	imagePullSecrets := []string{"imgpull-a", "imgpull-b"}

	branchName := "incoming"

	deploymentRepoManager, err := deploymentrepo.NewDeploymentRepoManager(
		ctfIn+"//github.com/openmcp-project/openmcp",
		"deployment-templates/templates",
		originDir,
		branchName,
		"./testdata/01/git-config.yaml").
		WithClusterProviders(clusterProviders).
		WithServiceProviders(serviceProviders).
		WithPlatformServices(platformServices).
		WithImagePullSecrets(imagePullSecrets).
		Complete(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, deploymentRepoManager)

	repo := deploymentRepoManager.GitRepoDir()

	err = deploymentRepoManager.ApplyTemplates(t.Context())
	assert.NoError(t, err)

	kustomization, err := deploymentrepo.ParseKustomization(filepath.Join(repo, "envs", branchName, "kustomization.yaml"))
	assert.NoError(t, err)
	assert.NotNil(t, kustomization)
	assert.Contains(t, kustomization.Resources, "../resources")
	assert.Contains(t, kustomization.Images, kimage.Image{
		Name:    "<openmcp/openmcp-operator>",
		NewName: "ghcr.io/openmcp-project/images/openmcp-operator",
		NewTag:  "v0.2.1",
	})

	err = deploymentRepoManager.ApplyProviders(t.Context())
	assert.NoError(t, err)

	clusterProviderTestRaw := testutils.ReadFromFile(t, filepath.Join(repo, "resources", "cluster-providers", "test.yaml"))
	var clusterProviderTest map[string]interface{}
	err = yaml.Unmarshal([]byte(clusterProviderTestRaw), &clusterProviderTest)
	assert.NoError(t, err)
	ValidateProvider(t, clusterProviderTest, "test", "ghcr.io/openmcp-project/images/cluster-provider-test:v0.1.0", []string{"imgpull-a", "imgpull-b"})

	serviceProviderTestRaw := testutils.ReadFromFile(t, filepath.Join(repo, "resources", "service-providers", "test.yaml"))
	var serviceProviderTest map[string]interface{}
	err = yaml.Unmarshal([]byte(serviceProviderTestRaw), &serviceProviderTest)
	assert.NoError(t, err)
	ValidateProvider(t, serviceProviderTest, "test", "ghcr.io/openmcp-project/images/service-provider-test:v0.2.0", []string{"imgpull-a", "imgpull-b"})

	platformServiceTestRaw := testutils.ReadFromFile(t, filepath.Join(repo, "resources", "platform-services", "test.yaml"))
	var platformServiceTest map[string]interface{}
	err = yaml.Unmarshal([]byte(platformServiceTestRaw), &platformServiceTest)
	assert.NoError(t, err)
	ValidateProvider(t, platformServiceTest, "test", "ghcr.io/openmcp-project/images/platform-service-test:v0.3.0", []string{"imgpull-a", "imgpull-b"})

	err = deploymentRepoManager.ApplyCustomResourceDefinitions(t.Context())
	assert.NoError(t, err)

	err = deploymentRepoManager.UpdateResourcesKustomization()
	assert.NoError(t, err)

	crdRaw := testutils.ReadFromFile(t, filepath.Join(repo, deploymentrepo.RepoResourcesDir, deploymentrepo.RepoCRDsDir, "crd.yaml"))
	assert.NotEmpty(t, crdRaw)

	err = deploymentRepoManager.CommitAndPushChanges(t.Context())
	assert.NoError(t, err)

	err = originWorkTree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branchName)})
	assert.NoError(t, err)

	envsDir := filepath.Join(originDir, deploymentrepo.RepoEnvsDir, branchName)
	resourcesDir := filepath.Join(originDir, deploymentrepo.RepoResourcesDir)

	_ = testutils.ReadFromFile(t, filepath.Join(originDir, "dummy.txt"))
	_ = testutils.ReadFromFile(t, filepath.Join(envsDir, "kustomization.yaml"))
	_ = testutils.ReadFromFile(t, filepath.Join(resourcesDir, "cluster-providers", "test.yaml"))
	_ = testutils.ReadFromFile(t, filepath.Join(resourcesDir, "service-providers", "test.yaml"))
	_ = testutils.ReadFromFile(t, filepath.Join(resourcesDir, "platform-services", "test.yaml"))
	_ = testutils.ReadFromFile(t, filepath.Join(resourcesDir, "crds", "crd.yaml"))
}
