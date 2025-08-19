package deploymentrepo_test

import (
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	deploymentrepo "github.com/openmcp-project/bootstrapper/internal/deployment-repo"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	testutils "github.com/openmcp-project/bootstrapper/test/utils"
)

func TestTemplateDir(t *testing.T) {
	templateDir := t.TempDir()
	repoDir := t.TempDir()

	templateFilePath := filepath.Join(templateDir, "template.txt")
	templateContent := "Hello, {{.values.name}}!"
	testutils.WriteToFile(t, templateFilePath, templateContent)

	repo, err := git.PlainInit(repoDir, false)
	assert.NoError(t, err)
	assert.NotNil(t, repo)

	templateInput := map[string]interface{}{
		"name": "World",
	}

	err = deploymentrepo.TemplateDir(templateDir, templateInput, repo)
	assert.NoError(t, err)

	templateResult := testutils.ReadFromFile(t, filepath.Join(repoDir, "template.txt"))
	assert.Equal(t, "Hello, World!", templateResult)

	workTree, err := repo.Worktree()
	assert.NoError(t, err)
	workTreeStatus, err := workTree.Status()
	assert.NoError(t, err)
	assert.False(t, workTreeStatus.IsClean())
	workTreeStatus.File("template.txt").Staging = git.Added
}

func TestTemplateProviders(t *testing.T) {
	testutils.DownloadOCMAndAddToPath(t)

	ctfIn := testutils.BuildComponent("./testdata/01/component-constructor.yaml", t)

	compGetter := ocmcli.NewComponentGetter(ctfIn+"//github.com/openmcp-project/openmcp", "deployment-templates/templates", ocmcli.NoOcmConfig)
	assert.NotNil(t, compGetter)

	err := compGetter.InitializeComponents(t.Context())
	assert.NoError(t, err)

	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	assert.NoError(t, err)

	clusterProviders := []string{"test"}
	serviceProviders := []string{"test"}
	platformServices := []string{"test"}
	imagePullSecrets := []string{"imgpull-a", "imgpull-b"}

	err = deploymentrepo.TemplateProviders(t.Context(), clusterProviders, serviceProviders, platformServices, imagePullSecrets, compGetter, repo)
	assert.NoError(t, err)

	clusterProviderTestRaw := testutils.ReadFromFile(t, filepath.Join(repoDir, "cluster-providers", "test.yaml"))
	var clusterProviderTest map[string]interface{}
	err = yaml.Unmarshal([]byte(clusterProviderTestRaw), &clusterProviderTest)
	assert.NoError(t, err)
	ValidateProvider(t, clusterProviderTest, "test", "ghcr.io/openmcp-project/images/cluster-provider-test:v0.1.0", []string{"imgpull-a", "imgpull-b"})

	serviceProviderTestRaw := testutils.ReadFromFile(t, filepath.Join(repoDir, "service-providers", "test.yaml"))
	var serviceProviderTest map[string]interface{}
	err = yaml.Unmarshal([]byte(serviceProviderTestRaw), &serviceProviderTest)
	assert.NoError(t, err)
	ValidateProvider(t, serviceProviderTest, "test", "ghcr.io/openmcp-project/images/service-provider-test:v0.2.0", []string{"imgpull-a", "imgpull-b"})

	platformServiceTestRaw := testutils.ReadFromFile(t, filepath.Join(repoDir, "platform-services", "test.yaml"))
	var platformServiceTest map[string]interface{}
	err = yaml.Unmarshal([]byte(platformServiceTestRaw), &platformServiceTest)
	assert.NoError(t, err)
	ValidateProvider(t, platformServiceTest, "test", "ghcr.io/openmcp-project/images/platform-service-test:v0.3.0", []string{"imgpull-a", "imgpull-b"})
}

func ValidateProvider(t *testing.T, provider map[string]interface{}, name, image string, imagePullSecrets []string) {
	assert.Contains(t, provider, "metadata")
	assert.Contains(t, provider["metadata"], "name")
	assert.Equal(t, name, provider["metadata"].(map[string]interface{})["name"])

	assert.Contains(t, provider, "spec")
	assert.Contains(t, provider["spec"], "image")
	assert.Equal(t, image, provider["spec"].(map[string]interface{})["image"])
	assert.Contains(t, provider["spec"], "imagePullSecrets")

	imagePullSecretsList := provider["spec"].(map[string]interface{})["imagePullSecrets"].([]interface{})
	assert.Len(t, imagePullSecretsList, len(imagePullSecrets))
	for _, ips := range imagePullSecrets {
		assert.Contains(t, imagePullSecretsList, map[string]interface{}{"name": ips})
	}
}
