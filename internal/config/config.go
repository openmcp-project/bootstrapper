package config

import (
	"encoding/json"
	"os"

	"github.com/fluxcd/pkg/apis/meta"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

type BootstrapperConfig struct {
	Component            Component              `json:"component"`
	DeploymentRepository DeploymentRepository   `json:"repository"`
	Providers            Providers              `json:"providers"`
	ImagePullSecrets     []string               `json:"imagePullSecrets"`
	OpenMCPOperator      OpenMCPOperator        `json:"openmcpOperator"`
	Environment          string                 `json:"environment"`
	TemplateInput        map[string]interface{} `json:"templateInput"`
	ExternalSecrets      ExternalSecrets        `json:"externalSecrets"`
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
	ClusterProviders []Provider `json:"clusterProviders"`
	ServiceProviders []Provider `json:"serviceProviders"`
	PlatformServices []Provider `json:"platformServices"`
}

type Provider struct {
	Name         string          `json:"name"`
	Config       json.RawMessage `json:"config"`
	ConfigParsed map[string]interface{}
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

type ExternalSecrets struct {
	RepositorySecretRef meta.LocalObjectReference   `json:"repositorySecretRef"`
	ImagePullSecrets    []meta.LocalObjectReference `json:"imagePullSecrets"`
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

	for i, cp := range c.Providers.ClusterProviders {
		if len(cp.Name) == 0 {
			errs = append(errs, field.Required(field.NewPath("providers.clusterProviders").Index(i).Child("name"), "cluster provider name is required"))
		}

		if cp.Config != nil {
			err := yaml.Unmarshal(cp.Config, &c.Providers.ClusterProviders[i].ConfigParsed)
			if err != nil {
				errs = append(errs, field.Invalid(field.NewPath("providers.clusterProviders").Index(i).Child("config"), string(cp.Config), "cluster provider config is not valid yaml"))
			}
		}
	}

	for i, sp := range c.Providers.ServiceProviders {
		if len(sp.Name) == 0 {
			errs = append(errs, field.Required(field.NewPath("providers.serviceProviders").Index(i).Child("name"), "service provider name is required"))
		}

		if sp.Config != nil {
			err := yaml.Unmarshal(sp.Config, &c.Providers.ServiceProviders[i].ConfigParsed)
			if err != nil {
				errs = append(errs, field.Invalid(field.NewPath("providers.serviceProviders").Index(i).Child("config"), string(sp.Config), "service provider config is not valid yaml"))
			}
		}
	}

	for i, ps := range c.Providers.PlatformServices {
		if len(ps.Name) == 0 {
			errs = append(errs, field.Required(field.NewPath("providers.platformServices").Index(i).Child("name"), "platform service name is required"))
		}

		if ps.Config != nil {
			err := yaml.Unmarshal(ps.Config, &c.Providers.PlatformServices[i].ConfigParsed)
			if err != nil {
				errs = append(errs, field.Invalid(field.NewPath("providers.platformServices").Index(i).Child("config"), string(ps.Config), "platform service config is not valid yaml"))
			}
		}
	}

	return errs.ToAggregate()
}
