package config

import (
	"encoding/json"
	"os"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

type BootstrapperConfig struct {
	Component            Component            `json:"component"`
	DeploymentRepository DeploymentRepository `json:"repository"`
	Providers            Providers            `json:"providers"`
	ImagePullSecrets     []string             `json:"imagePullSecrets"`
	OpenMCPOperator      OpenMCPOperator      `json:"openmcpOperator"`
	Environment          string               `json:"environment"`
}

type Component struct {
	OpenMCPComponentLocation            string `json:"location"`
	OpenMCPOperatorTemplateResourcePath string `json:"openmcpOperatorTemplateResourcePath"`
	FluxcdTemplateResourcePath          string `json:"fluxcdTemplateResourcePath"`
}

type DeploymentRepository struct {
	RepoURL    string `json:"url"`
	RepoBranch string `json:"branch"`
}

type TargetCluster struct {
	KubeconfigPath string `json:"kubeconfigPath"`
}

type Providers struct {
	ClusterProviders []string `json:"clusterProviders"`
	ServiceProviders []string `json:"serviceProviders"`
	PlatformServices []string `json:"platformServices"`
}

type OpenMCPOperator struct {
	Config       json.RawMessage `json:"config"`
	ConfigParsed map[string]interface{}
}

type Manifest struct {
	Name           string          `json:"name"`
	Manifest       json.RawMessage `json:"manifest"`
	ManifestParsed map[string]interface{}
}

func (c *BootstrapperConfig) ReadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, c)
}

func (c *BootstrapperConfig) SetDefaults() {
	if len(c.Component.FluxcdTemplateResourcePath) == 0 {
		c.Component.FluxcdTemplateResourcePath = "gitops-templates/fluxcd"
	}

	if len(c.Component.OpenMCPOperatorTemplateResourcePath) == 0 {
		c.Component.OpenMCPOperatorTemplateResourcePath = "gitops-templates/openmcp"
	}
}

func (c *BootstrapperConfig) Validate() error {
	errs := field.ErrorList{}

	if len(c.Environment) == 0 {
		errs = append(errs, field.Required(field.NewPath("environment"), "environment is required"))
	}

	if len(c.Component.OpenMCPComponentLocation) == 0 {
		errs = append(errs, field.Required(field.NewPath("component.location"), "component location is required"))
	}

	if len(c.DeploymentRepository.RepoURL) == 0 {
		errs = append(errs, field.Required(field.NewPath("repository.url"), "repository url is required"))
	}

	if len(c.DeploymentRepository.RepoBranch) == 0 {
		errs = append(errs, field.Required(field.NewPath("repository.branch"), "repository branch is required"))
	}

	if len(c.OpenMCPOperator.Config) == 0 {
		errs = append(errs, field.Required(field.NewPath("openmcpOperator.config"), "openmcp operator config is required"))
	}

	err := yaml.Unmarshal(c.OpenMCPOperator.Config, &c.OpenMCPOperator.ConfigParsed)
	if err != nil {
		errs = append(errs, field.Invalid(field.NewPath("openmcpOperator.config"), string(c.OpenMCPOperator.Config), "openmcp operator config is not valid yaml"))
	}

	return errs.ToAggregate()
}
