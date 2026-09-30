package ocm_cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"sigs.k8s.io/yaml"
)

const (
	// NoOcmConfig is a constant to indicate that no OCM configuration file is being provided.
	NoOcmConfig = ""

	// flagOutput is the OCM CLI flag selecting the output format or output path.
	flagOutput = "--output"
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

	// Only stdout carries the command result; the OCM CLI v2 writes its logs to stderr.
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ocm", ocmArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error executing ocm command: %w, %q", err, stderr.String())
	}

	return stdout.Bytes(), nil
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

// ListComponentVersions lists all versions of the component of cv in its repository, sorted ascending by semver.
func (cv *ComponentVersion) ListComponentVersions(ctx context.Context, ocmConfig string) ([]string, error) {
	out, err := ExecuteOutput(ctx, []string{"get", "componentversions", cv.Repository + "//" + cv.Component.Name}, []string{flagOutput, "yaml"}, ocmConfig)
	if err != nil {
		return nil, err
	}

	var cvs []ComponentVersion
	if err = yaml.Unmarshal(out, &cvs); err != nil {
		return nil, fmt.Errorf("error decoding component version list: %w", err)
	}

	versions := make([]*semver.Version, 0, len(cvs))
	for _, entry := range cvs {
		v, err := semver.NewVersion(entry.Component.Version)
		if err != nil {
			return nil, fmt.Errorf("error parsing component version %q: %w", entry.Component.Version, err)
		}
		versions = append(versions, v)
	}
	// The OCM CLI v2 lists versions in descending order; callers expect ascending order.
	slices.SortFunc(versions, func(a, b *semver.Version) int { return a.Compare(b) })

	cvList := make([]string, 0, len(versions))
	for _, v := range versions {
		cvList = append(cvList, v.Original())
	}
	return cvList, nil
}

// cvCache memoizes GetComponentVersion results for the lifetime of the process.
// Templating resolves the same release-channel / service / provider components
// repeatedly (once per extra-manifest template, across every environment), and
// each miss shells out to `ocm get componentversion` + a remote registry round-trip.
// Keyed by ocmConfig + componentReference; stores values so callers get a fresh
// pointer copy and cannot mutate shared cache state. Errors are not cached.
// ponytail: no singleflight — templating is serial, a cold-start double-exec is harmless.
var cvCache sync.Map

// GetComponentVersion retrieves a component version by its reference using the OCM CLI.
func GetComponentVersion(ctx context.Context, componentReference string, ocmConfig string) (*ComponentVersion, error) {
	cacheKey := ocmConfig + "\x00" + componentReference
	if cached, ok := cvCache.Load(cacheKey); ok {
		cv := cached.(ComponentVersion) // copy out — caller gets its own pointer
		return &cv, nil
	}

	out, err := ExecuteOutput(ctx, []string{"get", "componentversion", componentReference}, []string{flagOutput, "yaml"}, ocmConfig)
	if err != nil {
		return nil, err
	}

	// The OCM CLI v2 always prints a list, even for a single version-pinned reference.
	var cvs []ComponentVersion
	if err = yaml.Unmarshal(out, &cvs); err != nil {
		return nil, fmt.Errorf("error unmarshalling component version: %w", err)
	}
	if len(cvs) != 1 {
		return nil, fmt.Errorf("expected exactly one component version for %s, got %d", componentReference, len(cvs))
	}
	cv := cvs[0]

	cv.Repository = strings.SplitN(componentReference, "//", 2)[0]

	cvCache.Store(cacheKey, cv)

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
