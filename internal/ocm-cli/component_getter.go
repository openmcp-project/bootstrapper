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
	repo, err := extractRepoFromLocation(g.rootComponentLocation)
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
		cv, err = getReferencedComponentVersion(ctx, repo, cv, refName, g.ocmConfig)
		if err != nil {
			return fmt.Errorf("error getting referenced component version %s: %w", refName, err)
		}
	}

	g.templatesComponentVersion = cv
	g.templatesComponentLocation = buildLocation(repo, cv.Component.Name, cv.Component.Version)
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

func getReferencedComponentVersion(ctx context.Context, repo string, parentCV *ComponentVersion, refName string, ocmConfig string) (*ComponentVersion, error) {
	ref, err := parentCV.GetComponentReference(refName)
	if err != nil {
		return nil, fmt.Errorf("error getting component reference %s: %w", refName, err)
	}

	location := buildLocation(repo, ref.ComponentName, ref.Version)
	cv, err := GetComponentVersion(ctx, location, ocmConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting component version %s: %w", location, err)
	}

	return cv, nil
}

func (g *ComponentGetter) DownloadTemplatesResource(ctx context.Context, downloadDir string) error {
	return downloadDirectoryResource(ctx, g.templatesComponentLocation, g.templatesResourceName, downloadDir, g.ocmConfig)
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
