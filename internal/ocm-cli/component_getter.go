package ocm_cli

import (
	"context"
	"fmt"
	"strings"
)

type ComponentGetter struct {
	// Location of the root component in the format <repo>//<component>:<version>.
	rootComponentLocation string
	// Path to the deployment templates resource in the format <componentRef1>/.../<componentRefN>/<resourceName>.
	deploymentTemplates string
	ocmConfig           string

	// Fields derived during InitializeComponents
	repo string

	rootComponentVersion       *ComponentVersion
	templatesComponentVersion  *ComponentVersion
	templatesComponentLocation string
	templatesResourceName      string
}

func NewComponentGetter(rootComponentLocation, deploymentTemplates, ocmConfig string) *ComponentGetter {
	return &ComponentGetter{
		rootComponentLocation: rootComponentLocation,
		deploymentTemplates:   deploymentTemplates,
		ocmConfig:             ocmConfig,
	}
}

func (g *ComponentGetter) InitializeComponents(ctx context.Context) error {
	var err error

	g.repo, err = extractRepoFromLocation(g.rootComponentLocation)
	if err != nil {
		return err
	}

	rootComponentVersion, err := GetComponentVersion(ctx, g.rootComponentLocation, g.ocmConfig)
	if err != nil {
		return fmt.Errorf("error getting root component version %s: %w", g.rootComponentLocation, err)
	}
	g.rootComponentVersion = rootComponentVersion

	g.deploymentTemplates = strings.TrimSpace(g.deploymentTemplates)
	segments := strings.Split(g.deploymentTemplates, "/")
	if len(segments) == 0 {
		return fmt.Errorf("deploymentTemplates path must contain a resource name or component references and a resource name separated by slashes (ref1/.../refN/resource): %s", g.deploymentTemplates)
	}
	referenceNames := segments[:len(segments)-1]
	g.templatesResourceName = segments[len(segments)-1]

	cv := g.rootComponentVersion
	for _, refName := range referenceNames {
		cv, err = g.GetReferencedComponentVersionRecursive(ctx, cv, refName)
		if err != nil {
			return fmt.Errorf("error getting referenced component version %s: %w", refName, err)
		}
	}

	g.templatesComponentVersion = cv
	g.templatesComponentLocation = buildLocation(g.repo, cv.Component.Name, cv.Component.Version)
	return nil
}

func (g *ComponentGetter) RootComponentVersion() *ComponentVersion {
	return g.rootComponentVersion
}

func (g *ComponentGetter) TemplatesComponentVersion() *ComponentVersion {
	return g.templatesComponentVersion
}

func (g *ComponentGetter) TemplatesResourceName() string {
	return g.templatesResourceName
}

func (g *ComponentGetter) Repository() string {
	return g.repo
}

func (g *ComponentGetter) OCMConfig() string {
	return g.ocmConfig
}

func (g *ComponentGetter) GetReferencedComponentVersion(ctx context.Context, parentCV *ComponentVersion, refName string) (*ComponentVersion, error) {
	ref, err := parentCV.GetComponentReference(refName)
	if err != nil {
		return nil, fmt.Errorf("error getting component reference %s: %w", refName, err)
	}

	location := buildLocation(g.repo, ref.ComponentName, ref.Version)
	cv, err := GetComponentVersion(ctx, location, g.ocmConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting component version %s: %w", location, err)
	}

	return cv, nil
}

func (g *ComponentGetter) GetReferencedComponentVersionRecursive(ctx context.Context, parentCV *ComponentVersion, refName string) (*ComponentVersion, error) {
	// First, try to get the reference directly from the parent component version
	ref, err := g.GetReferencedComponentVersion(ctx, parentCV, refName)
	if err == nil {
		return ref, nil
	}

	// If not found, search recursively in all component references
	for _, componentRef := range parentCV.Component.ComponentReferences {
		subCV, err := g.GetReferencedComponentVersion(ctx, parentCV, componentRef.Name)
		if err != nil {
			continue
		}
		ref, err := g.GetReferencedComponentVersionRecursive(ctx, subCV, refName)
		if err == nil {
			return ref, nil
		}
	}

	return nil, fmt.Errorf("component reference %s not found in component version %s or its references", refName, parentCV.Component.Name)
}

func (g *ComponentGetter) GetComponentVersionForResourceRecursive(ctx context.Context, parentCV *ComponentVersion, resourceName string) (*ComponentVersion, error) {
	// Check if the resource exists in the current component version
	_, err := parentCV.GetResource(resourceName)
	if err == nil {
		return parentCV, nil
	}

	// If not found, search recursively in all component references
	for _, componentRef := range parentCV.Component.ComponentReferences {
		subCV, err := g.GetReferencedComponentVersion(ctx, parentCV, componentRef.Name)
		if err != nil {
			continue
		}
		cv, err := g.GetComponentVersionForResourceRecursive(ctx, subCV, resourceName)
		if err == nil {
			return cv, nil
		}
	}

	return nil, fmt.Errorf("resource %s not found in component version %s or its references", resourceName, parentCV.Component.Name)
}

func (g *ComponentGetter) DownloadTemplatesResource(ctx context.Context, downloadDir string) error {
	return downloadDirectoryResource(ctx, g.templatesComponentLocation, g.templatesResourceName, downloadDir, g.ocmConfig)
}

func (g *ComponentGetter) DownloadDirectoryResourceByLocation(ctx context.Context, rootCV *ComponentVersion, location string, downloadDir string) error {
	var err error

	location = strings.TrimSpace(location)
	segments := strings.Split(location, "/")
	if len(segments) == 0 {
		return fmt.Errorf("location must contain a resource name or component references and a resource name separated by slashes (ref1/.../refN/resource): %s", location)
	}

	referenceNames := segments[:len(segments)-1]
	resourceName := segments[len(segments)-1]

	cv := rootCV
	for _, refName := range referenceNames {
		cv, err = g.GetReferencedComponentVersionRecursive(ctx, cv, refName)
		if err != nil {
			return fmt.Errorf("error getting referenced component version %s: %w", refName, err)
		}
	}

	componentLocation := buildLocation(g.repo, cv.Component.Name, cv.Component.Version)
	return downloadDirectoryResource(ctx, componentLocation, resourceName, downloadDir, g.ocmConfig)
}

func (g *ComponentGetter) DownloadDirectoryResource(ctx context.Context, cv *ComponentVersion, resourceName string, downloadDir string) error {
	componentLocation := buildLocation(g.repo, cv.Component.Name, cv.Component.Version)
	return downloadDirectoryResource(ctx, componentLocation, resourceName, downloadDir, g.ocmConfig)
}

func downloadDirectoryResource(ctx context.Context, componentLocation string, resourceName string, downloadDir string, ocmConfig string) error {
	return Execute(ctx,
		[]string{"download", "resources", componentLocation, resourceName},
		[]string{"--downloader", "ocm/dirtree", "--outfile", downloadDir},
		ocmConfig,
	)
}

func extractRepoFromLocation(location string) (string, error) {
	parts := strings.SplitN(location, "//", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid component location format, expected '<repo>//<component>:<version>': %s", location)
	}
	return parts[0], nil
}

func buildLocation(repo, name, version string) string {
	return fmt.Sprintf("%s//%s:%s", repo, name, version)
}
