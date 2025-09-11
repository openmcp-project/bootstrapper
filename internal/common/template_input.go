package common

import (
	"fmt"

	"github.com/openmcp-project/bootstrapper/internal/config"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

type TemplateInput map[string]any

func NewTemplateInput() TemplateInput {
	return make(TemplateInput)
}

func NewTemplateInputFromConfig(c *config.BootstrapperConfig) TemplateInput {
	t := TemplateInput{
		"fluxCDEnvPath":       "./" + EnvsDirectoryName + "/" + c.Environment + "/" + FluxCDDirectoryName,
		"fluxCDResourcesPath": "../../../" + ResourcesDirectoryName + "/" + FluxCDDirectoryName,

		"git": map[string]interface{}{
			"repoUrl":    c.DeploymentRepository.RepoURL,
			"mainBranch": c.DeploymentRepository.RepoBranch,
		},
		"gitRepoEnvBranch": c.DeploymentRepository.RepoBranch,

		"imagePullSecrets": func() []map[string]string {
			if len(c.ImagePullSecrets) == 0 {
				return nil
			}
			secrets := make([]map[string]string, 0, len(c.ImagePullSecrets))
			for _, secret := range c.ImagePullSecrets {
				secrets = append(secrets, map[string]string{
					"name": secret,
				})
			}
			return secrets
		}(),
	}

	return t
}

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

func (t TemplateInput) ValuesWrapper() map[string]any {
	return map[string]any{
		"Values": t,
	}
}
