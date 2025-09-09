package template

import (
	"fmt"

	"github.com/openmcp-project/bootstrapper/internal/config"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

func NewTemplateInput() TemplateInput {
	return make(TemplateInput)
}

type TemplateInput map[string]any

func (t TemplateInput) AddImageResource(cv *ocmcli.ComponentVersion, resourceName, key string) error {
	resource, err := cv.GetResource(resourceName)
	if err != nil {
		return fmt.Errorf("failed to get resource %s: %w", resourceName, err)
	}
	imageName, imageTag, imageDigest, err := util.ParseImageVersionAndTag(*resource.Access.ImageReference)
	if err != nil {
		return fmt.Errorf("failed to parse image reference %s: %w", *resource.Access.ImageReference, err)
	}

	if _, found := t["images"]; !found {
		t["images"] = make(map[string]any)
	}
	t["images"].(map[string]any)[key] = map[string]any{
		"version": imageTag,
		"image":   imageName,
		"tag":     imageTag,
		"digest":  imageDigest,
	}
	return nil
}

func (t TemplateInput) SetImagePullSecrets(imagePullSecrets []string) {
	if len(imagePullSecrets) == 0 {
		return
	}

	t["imagePullSecrets"] = make([]map[string]string, 0, len(imagePullSecrets))
	for _, secret := range imagePullSecrets {
		t["imagePullSecrets"] = append(t["imagePullSecrets"].([]map[string]string), map[string]string{
			"name": secret,
		})
	}
}

func (t TemplateInput) SetGitRepo(repo config.DeploymentRepository) {
	t["git"] = map[string]interface{}{
		"repoUrl":    repo.RepoURL,
		"mainBranch": repo.RepoBranch,
	}
}
