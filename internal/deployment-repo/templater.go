package deploymentrepo

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"

	"github.com/openmcp-project/bootstrapper/internal/log"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/template"
)

// TemplateDir processes the template files in the specified directory and writes
// the rendered content to the corresponding files in the Git repository's worktree.
// It uses the provided template directory and Git repository to perform the operations.
func TemplateDir(templateDirectory string, templateInput map[string]interface{}, repo *git.Repository) error {
	logger := log.GetLogger()

	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	templateDir, err := os.Open(templateDirectory)
	if err != nil {
		return fmt.Errorf("failed to open template directory: %w", err)
	}
	defer func() {
		if err = templateDir.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close template directory: %v\n", err)
		}
	}()

	te := template.NewTemplateExecution().WithMissingKeyOption("zero")

	// Recursively walk through all files in the template directory
	err = filepath.WalkDir(templateDirectory, func(path string, d os.DirEntry, walkError error) error {
		var (
			errInWalk error

			templateFromFile []byte
			templateResult   []byte

			relativePath   string
			fileInWorkTree billy.File
		)

		if walkError != nil {
			return walkError
		}
		if !d.IsDir() {
			relativePath, errInWalk = filepath.Rel(templateDirectory, path)
			if errInWalk != nil {
				return fmt.Errorf("failed to get relative path for %s: %w", path, errInWalk)
			}

			logger.Debugf("Found template file: %s", relativePath)

			templateFromFile, errInWalk = os.ReadFile(path)
			if errInWalk != nil {
				return fmt.Errorf("failed to read template file %s: %w", relativePath, err)
			}

			wrappedTemplateInput := map[string]interface{}{
				"Values": templateInput,
			}

			templateResult, errInWalk = te.Execute(path, string(templateFromFile), wrappedTemplateInput)
			if errInWalk != nil {
				return fmt.Errorf("failed to execute template %s: %w", relativePath, errInWalk)
			}

			fileInWorkTree, errInWalk = workTree.Filesystem.OpenFile(relativePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if errInWalk != nil {
				return fmt.Errorf("failed to open file in worktree %s: %w", relativePath, errInWalk)
			}
			defer func(pathInRepo billy.File) {
				err := pathInRepo.Close()
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "failed to close file in worktree %s: %v\n", relativePath, err)
				}
			}(fileInWorkTree)

			_, errInWalk = fileInWorkTree.Write(templateResult)
			if errInWalk != nil {
				return fmt.Errorf("failed to write to file in worktree %s: %w", relativePath, errInWalk)
			}

			// Add the file to the git index
			if _, errInWalk = workTree.Add(relativePath); errInWalk != nil {
				return fmt.Errorf("failed to add file to git index: %w", errInWalk)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk template directory: %w", err)
	}

	return nil
}

var (
	//go:embed templates/clusterProvider.yaml
	clusterProviderTemplate string
	//go:embed templates/serviceProvider.yaml
	serviceProviderTemplate string
	//go:embed templates/platformService.yaml
	platformServiceTemplate string
)

type ProviderOptions struct {
	Name             string
	Image            string
	ImagePullSecrets []string
}

// TemplateProviders templates the specified cluster providers, service providers, and platform services
func TemplateProviders(ctx context.Context, clusterProviders, serviceProviders, platformServices, imagePullSecrets []string, ocmGetter *ocmcli.ComponentGetter, repo *git.Repository) error {
	basePath := filepath.Join("resources", "openmcp")
	clusterProvidersDir := filepath.Join(basePath, "cluster-providers")
	serviceProvidersDir := filepath.Join(basePath, "service-providers")
	platformServicesDir := filepath.Join(basePath, "platform-services")

	if _, err := os.Stat(clusterProvidersDir); err == nil {
		err = os.RemoveAll(clusterProvidersDir)
		if err != nil {
			return fmt.Errorf("failed to remove existing cluster providers directory: %w", err)
		}
	}

	if _, err := os.Stat(serviceProvidersDir); err == nil {
		err = os.RemoveAll(serviceProvidersDir)
		if err != nil {
			return fmt.Errorf("failed to remove existing service providers directory: %w", err)
		}
	}

	if _, err := os.Stat(platformServicesDir); err == nil {
		err = os.RemoveAll(platformServicesDir)
		if err != nil {
			return fmt.Errorf("failed to remove existing platform services directory: %w", err)
		}
	}

	for _, cp := range clusterProviders {
		componentVersion, err := ocmGetter.GetReferencedComponentVersionRecursive(ctx, ocmGetter.RootComponentVersion(), "cluster-provider-"+cp)
		if err != nil {
			return fmt.Errorf("failed to get component version for cluster provider %s: %w", cp, err)
		}

		imageResource, err := getImageResource(componentVersion)
		if err != nil {
			return fmt.Errorf("failed to get image resource for cluster provider %s: %w", cp, err)
		}

		opts := &ProviderOptions{
			Name:             cp,
			Image:            *imageResource.Access.ImageReference,
			ImagePullSecrets: imagePullSecrets,
		}

		err = templateProvider(opts, clusterProviderTemplate, clusterProvidersDir, repo)
		if err != nil {
			return fmt.Errorf("failed to apply cluster provider %s: %w", cp, err)
		}
	}

	for _, sp := range serviceProviders {
		componentVersion, err := ocmGetter.GetReferencedComponentVersionRecursive(ctx, ocmGetter.RootComponentVersion(), "service-provider-"+sp)
		if err != nil {
			return fmt.Errorf("failed to get component version for service provider %s: %w", sp, err)
		}

		imageResource, err := getImageResource(componentVersion)
		if err != nil {
			return fmt.Errorf("failed to get image resource for service provider %s: %w", sp, err)
		}

		opts := &ProviderOptions{
			Name:             sp,
			Image:            *imageResource.Access.ImageReference,
			ImagePullSecrets: imagePullSecrets,
		}

		err = templateProvider(opts, serviceProviderTemplate, serviceProvidersDir, repo)
		if err != nil {
			return fmt.Errorf("failed to apply service provider %s: %w", sp, err)
		}
	}

	for _, ps := range platformServices {
		componentVersion, err := ocmGetter.GetReferencedComponentVersionRecursive(ctx, ocmGetter.RootComponentVersion(), "platform-service-"+ps)
		if err != nil {
			return fmt.Errorf("failed to get component version for platform service %s: %w", ps, err)
		}

		imageResource, err := getImageResource(componentVersion)
		if err != nil {
			return fmt.Errorf("failed to get image resource for platform service %s: %w", ps, err)
		}

		opts := &ProviderOptions{
			Name:             ps,
			Image:            *imageResource.Access.ImageReference,
			ImagePullSecrets: imagePullSecrets,
		}

		err = templateProvider(opts, platformServiceTemplate, platformServicesDir, repo)
		if err != nil {
			return fmt.Errorf("failed to apply platform service %s: %w", ps, err)
		}
	}

	return nil
}

func getImageResource(cv *ocmcli.ComponentVersion) (*ocmcli.Resource, error) {
	resources := cv.GetResourcesByType(ocmcli.OCIImageResourceType)

	if len(resources) > 0 {
		return &resources[0], nil
	}

	return nil, fmt.Errorf("image resource not found for component %s", cv.Component.Name)
}

func templateProvider(options *ProviderOptions, templateSource, dir string, repo *git.Repository) error {
	logger := log.GetLogger()
	providerPath := filepath.Join(dir, options.Name+".yaml")

	logger.Debugf("Creating provider %s with image %s in path %s", options.Name, options.Image, providerPath)

	te := template.NewTemplateExecution()
	templateInput := map[string]interface{}{
		"values": map[string]interface{}{
			"name": options.Name,
			"image": map[string]interface{}{
				"location":         options.Image,
				"imagePullSecrets": options.ImagePullSecrets,
			},
		},
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	logger.Tracef("Template input: %v", templateInput)

	fileInWorkTree, err := workTree.Filesystem.OpenFile(providerPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s in worktree: %w", providerPath, err)
	}

	defer func(pathInRepo billy.File) {
		err := pathInRepo.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close file in worktree: %v\n", err)
		}
	}(fileInWorkTree)

	templateResult, err := te.Execute(providerPath, templateSource, templateInput)
	if err != nil {
		return fmt.Errorf("failed to execute templateSource for cluster provider %s: %w", options.Name, err)
	}

	_, err = fileInWorkTree.Write(templateResult)
	if err != nil {
		return fmt.Errorf("failed to write to file %s in worktree: %w", providerPath, err)
	}

	if _, err = workTree.Add(providerPath); err != nil {
		return fmt.Errorf("failed to add provider %s file to git index: %w", providerPath, err)
	}

	return nil
}
