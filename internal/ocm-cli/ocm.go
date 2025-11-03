package ocm_cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const (
	// NoOcmConfig is a constant to indicate that no OCM configuration file is being provided.
	NoOcmConfig = ""
)

// Execute runs the specified OCM command with the provided arguments and configuration.
// It captures the command's output and errors, and returns an error if the command fails.
// The `commands` parameter is a slice of strings representing the OCM command and its subcommands.
// The `args` parameter is a slice of strings representing the arguments to the command.
// The `ocmConfig` parameter is a string representing the path to the OCM configuration file. Passing `NoOcmConfig` indicates that no configuration file should be used.
func Execute(ctx context.Context, commands []string, args []string, ocmConfig string) error {
	var ocmArgs []string

	if ocmConfig != NoOcmConfig {
		ocmArgs = append(ocmArgs, "--config", ocmConfig)

		if err := verifyOCMConfig(ocmConfig); err != nil {
			return fmt.Errorf("invalid OCM configuration: %w", err)
		}
	}

	ocmArgs = append(ocmArgs, commands...)
	ocmArgs = append(ocmArgs, args...)

	cmd := exec.CommandContext(ctx, "ocm", ocmArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ocm command: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error waiting for ocm command to finish: %w", err)
	}

	if cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("ocm command exited with code %d", cmd.ProcessState.ExitCode())
	}

	return nil
}

func ExecuteOutput(ctx context.Context, commands []string, args []string, ocmConfig string) ([]byte, error) {
	var ocmArgs []string

	if ocmConfig != NoOcmConfig {
		ocmArgs = append(ocmArgs, "--config", ocmConfig)

		if err := verifyOCMConfig(ocmConfig); err != nil {
			return nil, fmt.Errorf("invalid OCM configuration: %w", err)
		}
	}

	ocmArgs = append(ocmArgs, commands...)
	ocmArgs = append(ocmArgs, args...)

	cmd := exec.CommandContext(ctx, "ocm", ocmArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing ocm command: %w, %q", err, out)
	}

	if cmd.ProcessState.ExitCode() != 0 {
		return nil, fmt.Errorf("ocm command exited with code %d: %q", cmd.ProcessState.ExitCode(), out)
	}

	return out, nil
}

// ComponentVersion represents a version of an OCM component.
type ComponentVersion struct {
	// Component is the OCM component associated with this version.
	Component  Component `json:"component"`
	Repository string    `json:"repository,omitempty"`
}

// Component represents an OCM component with its name, version, references to other components, and resources.
type Component struct {
	// Name is the name of the component.
	Name string `json:"name"`
	// Version is the version of the component.
	Version string `json:"version"`
	// ComponentReferences is a list of references to other components that this component depends on.
	ComponentReferences []ComponentReference `json:"componentReferences"`
	// Resources is a list of resources associated with this component, including their names, versions, types, and access information.
	Resources []Resource `json:"resources"`
}

// ComponentReference represents a reference to another component, including its name, version, and the name of the component it refers to.
type ComponentReference struct {
	// Name is the name of the component reference.
	Name string `json:"name"`
	// Version is the version of the component reference.
	Version string `json:"version"`
	// ComponentName is the name of the component that this reference points to.
	ComponentName string `json:"componentName"`
}

// Resource represents a resource associated with a component, including its name, version, type, and access information.
type Resource struct {
	// Name is the name of the resource.
	Name string `json:"name"`
	// Version is the version of the resource.
	Version string `json:"version"`
	// Type is the content type of the resource.
	Type string `json:"type"`
	// Access contains the information on how to access the resource.
	Access Access `json:"access"`
}

// Access represents the access information for a resource, including the type of access.
type Access struct {
	// Type specifies the access type of the resource.
	Type string `json:"type"`
	// ImageReference is the reference to the image if the Type is "ociArtifact".
	ImageReference *string `json:"imageReference"`
	// LocalReference specifies a component local access
	LocalReference *string `json:"localReference"`
	// MediaType is the media type of the resource
	MediaType *string `json:"mediaType"`
}

type ComponentListEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

var (
	OCIImageResourceType = "ociImage"
)

// GetResource retrieves a resource by its name from the component version.
func (cv *ComponentVersion) GetResource(name string) (*Resource, error) {
	for _, resource := range cv.Component.Resources {
		if resource.Name == name {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("resource %s not found in component version %s", name, cv.Component.Name)
}

func (cv *ComponentVersion) GetResourcesByType(resourceType string) []Resource {
	var resources []Resource
	for _, resource := range cv.Component.Resources {
		if resource.Type == resourceType {
			resources = append(resources, resource)
		}
	}
	return resources
}

// GetComponentReferences retrieves component references by its name from the component version.
func (cv *ComponentVersion) GetComponentReferences(name string) []ComponentReference {
	references := make([]ComponentReference, 0)

	for _, ref := range cv.Component.ComponentReferences {
		if ref.Name == name {
			references = append(references, ref)
		}
	}
	return references
}

func (cv *ComponentVersion) ListComponentVersions(ctx context.Context, ocmConfig string) ([]string, error) {

	out, err := ExecuteOutput(ctx, []string{"list", "componentversion", cv.Repository + "//" + cv.Component.Name}, []string{"--output", "yaml"}, ocmConfig)
	if err != nil {
		return nil, err
	}

	cvList := make([]string, 0)
	decoder := yaml2.NewYAMLOrJSONDecoder(strings.NewReader(string(out)), 1024)
	for {
		var entry ComponentListEntry
		err = decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error decoding component version list: %w", err)
		}
		cvList = append(cvList, entry.Version)
	}

	return cvList, nil

}

// GetComponentVersion retrieves a component version by its reference using the OCM CLI.
func GetComponentVersion(ctx context.Context, componentReference string, ocmConfig string) (*ComponentVersion, error) {
	out, err := ExecuteOutput(ctx, []string{"get", "componentversion", componentReference}, []string{"--output", "yaml"}, ocmConfig)
	if err != nil {
		return nil, err
	}

	var cv ComponentVersion
	err = yaml.Unmarshal(out, &cv)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling component version: %w", err)
	}

	cv.Repository = strings.SplitN(componentReference, "//", 2)[0]

	return &cv, nil
}

type OCMConfiguration struct {
	Type           string                        `json:"type"`
	Configurations []TypedOCMConfigConfiguration `json:"configurations"`
}

type TypedOCMConfigConfiguration struct {
	Type         string                `json:"type"`
	Repositories []OCMConfigRepository `json:"repositories"`
}

type OCMConfigRepository struct {
	Repository *TypedOCMConfigRepository `json:"repository"`
}

type TypedOCMConfigRepository struct {
	Type string `json:"type"`
}

const (
	DockerConfigRepositoryType = "DockerConfig"
)

// verifyOCMConfig checks if the OCM configuration file exists and does not contain unsupported configuration.
func verifyOCMConfig(ocmConfig string) error {
	if _, err := os.Stat(ocmConfig); os.IsNotExist(err) {
		return fmt.Errorf("OCM configuration file does not exist: %s", ocmConfig)
	}

	file, err := os.ReadFile(ocmConfig)
	if err != nil {
		return fmt.Errorf("error reading OCM configuration file: %w", err)
	}

	var config OCMConfiguration
	if err = yaml.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("error unmarshalling OCM configuration file: %w", err)
	}

	for _, typedConfig := range config.Configurations {
		for _, repo := range typedConfig.Repositories {
			if repo.Repository != nil && strings.Contains(repo.Repository.Type, DockerConfigRepositoryType) {
				return fmt.Errorf("unsupported repository type: %s", repo.Repository.Type)
			}
		}
	}

	return nil
}
