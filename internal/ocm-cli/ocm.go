package ocm_cli

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
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
	var flags []string

	flags = append(flags, commands...)
	flags = append(flags, args...)

	if ocmConfig != NoOcmConfig {
		flags = append(flags, "--config", ocmConfig)
	}

	cmd := exec.CommandContext(ctx, "ocm", flags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ocm command: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error waiting for ocm command to finish: %w", err)
	}

	return nil
}

// ComponentVersion represents a version of an OCM component.
type ComponentVersion struct {
	// Component is the OCM component associated with this version.
	Component Component `json:"component"`
}

// Component represents an OCM component with its name, version, references to other components, and resources.
type Component struct {
	// Name is the name of the component.
	Name string `yaml:"name"`
	// Version is the version of the component.
	Version string `yaml:"version"`
	// ComponentReferences is a list of references to other components that this component depends on.
	ComponentReferences []ComponentReference `yaml:"componentReferences"`
	// Resources is a list of resources associated with this component, including their names, versions, types, and access information.
	Resources []Resource `yaml:"resources"`
}

// ComponentReference represents a reference to another component, including its name, version, and the name of the component it refers to.
type ComponentReference struct {
	// Name is the name of the component reference.
	Name string `yaml:"name"`
	// Version is the version of the component reference.
	Version string `yaml:"version"`
	// ComponentName is the name of the component that this reference points to.
	ComponentName string `yaml:"componentName"`
}

// Resource represents a resource associated with a component, including its name, version, type, and access information.
type Resource struct {
	// Name is the name of the resource.
	Name string `yaml:"name"`
	// Version is the version of the resource.
	Version string `yaml:"version"`
	// Type is the content type of the resource.
	Type string `yaml:"type"`
	// Access contains the information on how to access the resource.
	Access Access `yaml:"access"`
}

// Access represents the access information for a resource, including the type of access.
type Access struct {
	// Type is the content type of access to the resource.
	Type string `yaml:"type"`
	// ImageReference is the reference to the image if the Type is "ociArtifact".
	ImageReference string `yaml:"imageReference"`
}

// GetResource retrieves a resource by its name from the component version.
func (cv *ComponentVersion) GetResource(name string) (*Resource, error) {
	for _, resource := range cv.Component.Resources {
		if resource.Name == name {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("resource %s not found in component version %s", name, cv.Component.Name)
}

// GetComponentReference retrieves a component reference by its name from the component version.
func (cv *ComponentVersion) GetComponentReference(name string) (*ComponentReference, error) {
	for _, ref := range cv.Component.ComponentReferences {
		if ref.Name == name {
			return &ref, nil
		}
	}
	return nil, fmt.Errorf("component reference %s not found in component version %s", name, cv.Component.Name)
}

// GetComponentVersion retrieves a component version by its reference using the OCM CLI.
func GetComponentVersion(ctx context.Context, componentReference string, ocmConfig string) (*ComponentVersion, error) {
	flags := []string{
		"get",
		"componentversion",
		"--output", "yaml",
		componentReference,
	}

	if ocmConfig != NoOcmConfig {
		flags = append(flags, "--config", ocmConfig)
	}

	cmd := exec.CommandContext(ctx, "ocm", flags...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing ocm command: %w", err)
	}

	var cv ComponentVersion
	err = yaml.Unmarshal(out, &cv)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling component version: %w", err)
	}

	return &cv, nil
}
