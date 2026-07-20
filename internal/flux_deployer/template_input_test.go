package flux_deployer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	"github.com/openmcp-project/bootstrapper/internal/config"
	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	ocm_cli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
)

func TestNewTemplateInputFromConfig(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		expectedProvider string
	}{
		{
			name:             "provider set to github",
			provider:         "github",
			expectedProvider: "github",
		},
		{
			name:             "provider unset passes through as empty",
			provider:         "",
			expectedProvider: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.BootstrapperConfig{
				Environment: "test-env",
				DeploymentRepository: config.DeploymentRepository{
					RepoURL:    "test-repo-url",
					PullBranch: "test-branch-pull",
					PushBranch: "test-branch-push",
					Provider:   tt.provider,
				},
				ImagePullSecrets: []string{"test-secret"},
			}

			ti := flux_deployer.NewTemplateInputFromConfig(cfg)
			assert.NotNil(t, ti, "Expected non-nil TemplateInput")
			assert.Equal(t, "./envs/test-env/fluxcd", ti["fluxCDEnvPath"], "fluxCDEnvPath does not match")
			assert.Equal(t, "../../../resources/fluxcd", ti["fluxCDResourcesPath"], "fluxCDResourcesPath does not match")
			assert.Equal(t, "../../../resources/openmcp", ti["openMCPResourcesPath"], "openMCPResourcesPath does not match")
			assert.Len(t, ti["imagePullSecrets"].([]map[string]string), 1, "imagePullSecrets length does not match")
			assert.Equal(t, "test-secret", ti["imagePullSecrets"].([]map[string]string)[0]["name"], "imagePullSecret name does not match")
			assert.Equal(t, "test-repo-url", ti["git"].(map[string]interface{})["repoUrl"], "git repoUrl does not match")
			assert.Equal(t, "test-branch-push", ti["git"].(map[string]interface{})["pushBranch"], "git pushBranch does not match")
			assert.Equal(t, "test-branch-pull", ti["git"].(map[string]interface{})["pullBranch"], "git pullBranch does not match")
			assert.Equal(t, tt.expectedProvider, ti["git"].(map[string]interface{})["provider"], "git provider does not match")
		})
	}
}

func TestTemplateInput_AddImageResource(t *testing.T) {
	cv := &ocm_cli.ComponentVersion{
		Component: ocm_cli.Component{
			Resources: []ocm_cli.Resource{
				{
					Name: "test-resource",
					Access: ocm_cli.Access{
						ImageReference: ptr.To("test-image:v1.0.0@sha256:123456789abcdef"),
					},
				},
			},
		},
	}

	ti := flux_deployer.TemplateInput{}
	err := ti.AddImageResource(cv, "test-resource", "testKey")
	assert.NoError(t, err, "Expected no error adding image resource")
	images, ok := ti["images"].(map[string]any)
	assert.True(t, ok, "Expected images to be a map")
	imageInfo, ok := images["testKey"].(map[string]any)
	assert.True(t, ok, "Expected imageInfo to be a map")
	assert.Equal(t, "test-image", imageInfo["image"], "Image name does not match")
	assert.Equal(t, "v1.0.0", imageInfo["tag"], "Image tag does not match")
	assert.Equal(t, "v1.0.0", imageInfo["version"], "Image version does not match")
	assert.Equal(t, "sha256:123456789abcdef", imageInfo["digest"], "Image digest does not match")
}
