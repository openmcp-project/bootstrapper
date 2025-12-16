package deploymentrepo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"

	"github.com/openmcp-project/bootstrapper/internal/config"

	gitconfig "github.com/openmcp-project/bootstrapper/internal/git-config"
	"github.com/openmcp-project/bootstrapper/internal/log"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	OpenMCPOperatorComponentName              = "openmcp-operator"
	FluxCDSourceControllerResourceName        = "fluxcd-source-controller"
	FluxCDKustomizationControllerResourceName = "fluxcd-kustomize-controller"
	FluxCDHelmControllerResourceName          = "fluxcd-helm-controller"
	FluxCDNotificationControllerName          = "fluxcd-notification-controller"
	FluxCDImageReflectorControllerName        = "fluxcd-image-reflector-controller"
	FluxCDImageAutomationControllerName       = "fluxcd-image-automation-controller"

	EnvsDirectoryName       = "envs"
	ResourcesDirectoryName  = "resources"
	OpenMCPDirectoryName    = "openmcp"
	FluxCDDirectoryName     = "fluxcd"
	CRDsDirectoryName       = "crds"
	ExtraManifestsDirectory = "extra"
)

// DeploymentRepoManager manages the deployment repository by applying templates and committing changes.
type DeploymentRepoManager struct {
	// Vars set via constructor or "With" methods

	// GitConfigPath is the path to the Git configuration file
	GitConfigPath string
	// OcmConfigPath is the path to the OCM configuration file
	// +optional
	OcmConfigPath string

	// Config is the bootstrapper configuration
	Config *config.BootstrapperConfig

	// TargetCluster is the Kubernetes cluster to which the deployment will be applied
	TargetCluster *clusters.Cluster

	// ExtraManifestDir is an optional directory containing extra manifests to be added to the deployment repository
	ExtraManifestDir string

	PatchesFile string

	// Internals
	// workDir is a temporary directory used for processing
	workDir string
	// templatesDir is the directory into which the templates resource are downloaded
	// (a subdirectory of workDir)
	templatesDir string
	// gitRepoDir is the directory into which the deployment repository is cloned
	// (a subdirectory of workDir)
	gitRepoDir string

	// compGetter is the OCM component getter used to fetch components and resources
	compGetter *ocmcli.ComponentGetter
	// gitConfig is the parsed Git configuration
	gitConfig *gitconfig.Config
	// gitRepo is the cloned Git repository
	gitRepo *git.Repository
	// openMCPOperatorCV is the component version of the openmcp-operator component
	openMCPOperatorCV *ocmcli.ComponentVersion
	// fluxcdCV is the component version of the fluxcd source controller component
	fluxcdCV *ocmcli.ComponentVersion
	// crdFiles is a list of CRD files downloaded from the openmcp-operator component
	crdFiles []string
	// extraManifests is a list of extra manifest files copied from the ExtraManifestDir to the deployment repository
	extraManifests []string
}

// NewDeploymentRepoManager creates a new DeploymentRepoManager with the specified parameters.
func NewDeploymentRepoManager(config *config.BootstrapperConfig, targetCluster *clusters.Cluster, gitConfigPath, ocmConfigPath, extraManifestDir, patchesFile string) *DeploymentRepoManager {
	return &DeploymentRepoManager{
		Config:           config,
		TargetCluster:    targetCluster,
		GitConfigPath:    gitConfigPath,
		OcmConfigPath:    ocmConfigPath,
		ExtraManifestDir: extraManifestDir,
		PatchesFile:      patchesFile,
	}
}

// Initialize initializes the DeploymentRepoManager by setting up working directories, downloading components and templates, and cloning the deployment repository.
func (m *DeploymentRepoManager) Initialize(ctx context.Context) (*DeploymentRepoManager, error) {
	var err error

	logger := log.GetLogger()

	m.workDir, err = util.CreateTempDir()
	if err != nil {
		return m, fmt.Errorf("failed to create working directory for deployment repository: %w", err)
	}

	logger.Tracef("Created working dir: %s", m.workDir)

	m.templatesDir = filepath.Join(m.workDir, "templates")
	m.gitRepoDir = filepath.Join(m.workDir, "repo")

	err = os.Mkdir(m.templatesDir, 0o755)
	if err != nil {
		return m, fmt.Errorf("failed to create working directory: %w", err)
	}

	logger.Tracef("Created template dir: %s", m.templatesDir)

	err = os.Mkdir(m.gitRepoDir, 0o755)
	if err != nil {
		return m, fmt.Errorf("failed to create template directory: %w", err)
	}

	logger.Tracef("Created Git repo dir: %s", m.gitRepoDir)

	logger.Infof("Downloading component %s", m.Config.Component.OpenMCPComponentLocation)

	m.compGetter = ocmcli.NewComponentGetter(m.Config.Component.OpenMCPComponentLocation, m.Config.Component.FluxcdTemplateResourcePath, m.OcmConfigPath)
	err = m.compGetter.InitializeComponents(ctx)
	if err != nil {
		return m, fmt.Errorf("failed to initialize components: %w", err)
	}

	logger.Info("Creating template transformer")

	templateTransformer := NewTemplateTransformer(m.compGetter, m.Config.Component.FluxcdTemplateResourcePath, m.Config.Component.OpenMCPOperatorTemplateResourcePath, m.workDir)
	err = templateTransformer.Transform(ctx, m.Config.Environment, m.templatesDir)
	if err != nil {
		return m, fmt.Errorf("failed to transform templates: %w", err)
	}

	logger.Infof("Fetching openmcp-operator component version")

	openMCPOperatorCVs, err := m.compGetter.GetReferencedComponentVersionsRecursive(ctx, m.compGetter.RootComponentVersion(), OpenMCPOperatorComponentName)
	if err != nil {
		return m, fmt.Errorf("failed to get openmcp-operator component version: %w", err)
	}
	if len(openMCPOperatorCVs) != 1 {
		return m, fmt.Errorf("expected exactly one openmcp-operator component version, got %d", len(openMCPOperatorCVs))
	}
	m.openMCPOperatorCV = &openMCPOperatorCVs[0]

	fluxcdCVs, err := m.compGetter.GetComponentVersionsForResourceRecursive(ctx, m.compGetter.RootComponentVersion(), FluxCDSourceControllerResourceName)
	if err != nil {
		return m, fmt.Errorf("failed to get fluxcd source controller component version: %w", err)
	}
	if len(fluxcdCVs) != 1 {
		return m, fmt.Errorf("expected exactly one fluxcd source controller component version, got %d", len(fluxcdCVs))
	}
	m.fluxcdCV = &fluxcdCVs[0]

	m.gitConfig, err = gitconfig.ParseConfig(m.GitConfigPath)
	if err != nil {
		return m, fmt.Errorf("failed to parse git config: %w", err)
	}
	err = m.gitConfig.Validate()
	if err != nil {
		return m, fmt.Errorf("invalid git config: %w", err)
	}

	logger.Infof("Cloning deployment repository %s", m.Config.DeploymentRepository.RepoURL)

	m.gitRepo, err = CloneRepo(m.Config.DeploymentRepository.RepoURL, m.gitRepoDir, m.gitConfig)
	if err != nil {
		return m, fmt.Errorf("failed to clone deployment repository: %w", err)
	}

	logger.Infof("Checking out or creating branch %s", m.Config.DeploymentRepository.RepoBranch)

	err = CheckoutAndCreateBranchIfNotExists(m.gitRepo, m.Config.DeploymentRepository.RepoBranch, m.gitConfig)
	if err != nil {
		return m, fmt.Errorf("failed to checkout or create branch %s: %w", m.Config.DeploymentRepository.RepoBranch, err)
	}

	return m, nil
}

// Cleanup removes temporary directories created during processing.
func (m *DeploymentRepoManager) Cleanup() {
	if m.workDir != "" {
		log.GetLogger().Tracef("Removing working dir: %s", m.workDir)
		if err := os.RemoveAll(m.workDir); err != nil {
			log.GetLogger().Warnf("Failed to working dir %s: %v", m.workDir, err)
		}
	}
}

// ApplyTemplates applies the templates from the templates directory to the deployment repository.
func (m *DeploymentRepoManager) ApplyTemplates(ctx context.Context) error {
	logger := log.GetLogger()

	logger.Infof("Applying templates from %q/%q to deployment repository", m.Config.Component.FluxcdTemplateResourcePath, m.Config.Component.OpenMCPOperatorTemplateResourcePath)
	templateInput := make(map[string]interface{})

	openMCPOperatorImageResources := m.openMCPOperatorCV.GetResourcesByType(ocmcli.OCIImageResourceType)
	if len(openMCPOperatorImageResources) == 0 || openMCPOperatorImageResources[0].Access.ImageReference == nil {
		return fmt.Errorf("no image resource found for openmcp-operator component version %s:%s", m.openMCPOperatorCV.Component.Name, m.openMCPOperatorCV.Component.Version)
	}

	imageName, imageTag, imageDigest, err := util.ParseImageVersionAndTag(*openMCPOperatorImageResources[0].Access.ImageReference)
	if err != nil {
		return fmt.Errorf("failed to parse image reference %s: %w", *openMCPOperatorImageResources[0].Access.ImageReference, err)
	}

	if len(m.Config.ImagePullSecrets) > 0 {
		templateInput["imagePullSecrets"] = make([]map[string]string, 0, len(m.Config.ImagePullSecrets))
		for _, secret := range m.Config.ImagePullSecrets {
			templateInput["imagePullSecrets"] = append(templateInput["imagePullSecrets"].([]map[string]string), map[string]string{
				"name": secret,
			})
		}
	}

	templateInput["openmcpOperator"] = map[string]interface{}{
		"version":          m.openMCPOperatorCV.Component.Version,
		"image":            imageName,
		"tag":              imageTag,
		"digest":           imageDigest,
		"imagePullSecrets": m.Config.ImagePullSecrets,
		"environment":      m.Config.Environment,
		"config":           m.Config.OpenMCPOperator.ConfigParsed,
	}

	templateInput["user"] = m.Config.TemplateInput
	templateInput["fluxCDEnvPath"] = "./" + EnvsDirectoryName + "/" + m.Config.Environment + "/" + FluxCDDirectoryName
	templateInput["gitRepoEnvBranch"] = m.Config.DeploymentRepository.RepoBranch
	templateInput["fluxCDResourcesPath"] = "../../../" + ResourcesDirectoryName + "/" + FluxCDDirectoryName
	templateInput["openMCPResourcesPath"] = "../../../" + ResourcesDirectoryName + "/" + OpenMCPDirectoryName
	templateInput["git"] = map[string]interface{}{
		"repoUrl":    m.Config.DeploymentRepository.RepoURL,
		"mainBranch": m.Config.DeploymentRepository.RepoBranch,
	}

	templateInput["images"] = make(map[string]interface{})

	if len(m.PatchesFile) > 0 {
		var userKustomizationPatches map[string]interface{}
		patchesRaw, err := os.ReadFile(m.PatchesFile)
		if err != nil {
			return fmt.Errorf("failed to read patches file %s: %w", m.PatchesFile, err)
		}

		patches, err := TemplateString(ctx, "userPatches", string(patchesRaw), templateInput, m.compGetter)
		if err != nil {
			return fmt.Errorf("failed to template user patches file %s: %w", m.PatchesFile, err)
		}

		err = yaml.Unmarshal([]byte(patches), &userKustomizationPatches)
		if err != nil {
			return fmt.Errorf("failed to unmarshal user patches from file %s: %w", m.PatchesFile, err)
		}

		if userKustomizationPatches["patches"] == nil {
			return fmt.Errorf("no patches found in user patches file %s", m.PatchesFile)
		}

		templateInput["userKustomizationPatches"] = userKustomizationPatches["patches"]
	}

	err = applyFluxCDTemplateInput(templateInput, m.fluxcdCV, FluxCDSourceControllerResourceName, "sourceController")
	if err != nil {
		return fmt.Errorf("failed to apply fluxcd source controller template input: %w", err)
	}

	err = applyFluxCDTemplateInput(templateInput, m.fluxcdCV, FluxCDKustomizationControllerResourceName, "kustomizeController")
	if err != nil {
		return fmt.Errorf("failed to apply fluxcd kustomize controller template input: %w", err)
	}

	err = applyFluxCDTemplateInput(templateInput, m.fluxcdCV, FluxCDHelmControllerResourceName, "helmController")
	if err != nil {
		return fmt.Errorf("failed to apply fluxcd helm controller template input: %w", err)
	}

	err = applyFluxCDTemplateInput(templateInput, m.fluxcdCV, FluxCDNotificationControllerName, "notificationController")
	if err != nil {
		return fmt.Errorf("failed to apply fluxcd helm controller template input: %w", err)
	}

	err = applyFluxCDTemplateInput(templateInput, m.fluxcdCV, FluxCDImageReflectorControllerName, "imageReflectorController")
	if err != nil {
		return fmt.Errorf("failed to apply fluxcd image reflector controller template input: %w", err)
	}

	err = applyFluxCDTemplateInput(templateInput, m.fluxcdCV, FluxCDImageAutomationControllerName, "imageAutomationController")
	if err != nil {
		return fmt.Errorf("failed to apply fluxcd image automation controller template input: %w", err)
	}

	if len(m.ExtraManifestDir) > 0 {
		err = util.CopyDir(m.ExtraManifestDir, filepath.Join(m.templatesDir, ResourcesDirectoryName, OpenMCPDirectoryName, ExtraManifestsDirectory))
		if err != nil {
			return fmt.Errorf("failed to copy extra manifests from %s to deployment repository: %w", m.ExtraManifestDir, err)
		}
	}

	err = TemplateDir(ctx, m.templatesDir, templateInput, m.compGetter, m.gitRepo)
	if err != nil {
		return fmt.Errorf("failed to apply templates from directory %s: %w", m.templatesDir, err)
	}

	return nil
}

func applyFluxCDTemplateInput(templateInput map[string]interface{}, fluxcdCV *ocmcli.ComponentVersion, fluxResource, key string) error {
	fluxSourceControllerImageResource, err := fluxcdCV.GetResource(fluxResource)
	if err != nil {
		return fmt.Errorf("failed to get fluxcd resource %s: %w", fluxResource, err)
	}
	imageName, imageTag, imageDigest, err := util.ParseImageVersionAndTag(*fluxSourceControllerImageResource.Access.ImageReference)
	if err != nil {
		return fmt.Errorf("failed to parse image reference %s: %w", *fluxSourceControllerImageResource.Access.ImageReference, err)
	}
	templateInput["images"].(map[string]interface{})[key] = map[string]interface{}{
		"version": imageTag,
		"image":   imageName,
		"tag":     imageTag,
		"digest":  imageDigest,
	}
	return nil
}

// ApplyProviders applies the specified providers to the deployment repository.
func (m *DeploymentRepoManager) ApplyProviders(ctx context.Context) error {
	logger := log.GetLogger()

	logger.Infof("Templating providers: clusterProviders=%v, serviceProviders=%v, platformServices=%v, imagePullSecrets=%v",
		m.Config.Providers.ClusterProviders, m.Config.Providers.ServiceProviders, m.Config.Providers.PlatformServices, m.Config.ImagePullSecrets)

	err := TemplateProviders(ctx, m.Config.Providers.ClusterProviders, m.Config.Providers.ServiceProviders, m.Config.Providers.PlatformServices, m.Config.ImagePullSecrets, m.compGetter, m.gitRepo)
	if err != nil {
		return fmt.Errorf("failed to template providers: %w", err)
	}

	return nil
}

// ApplyCustomResourceDefinitions downloads and applies Custom Resource Definitions (CRDs) from the openmcp-operator component to the deployment repository.
// If the openmcp-operator component is not found, it skips this step.
func (m *DeploymentRepoManager) ApplyCustomResourceDefinitions(ctx context.Context) error {
	logger := log.GetLogger()

	if m.openMCPOperatorCV == nil {
		logger.Infof("No openmcp-operator component version found, skipping CRD application")
		return nil
	}

	crdDirectory := filepath.Join(m.gitRepoDir, ResourcesDirectoryName, OpenMCPDirectoryName, CRDsDirectoryName)

	logger.Infof("Applying Custom Resource Definitions to deployment repository")

	err := m.applyCRDsForComponentVersion(ctx, m.openMCPOperatorCV, crdDirectory)
	if err != nil {
		return fmt.Errorf("failed to apply CRDs for openmcp-operator component: %w", err)
	}

	for _, clusterProvider := range m.Config.Providers.ClusterProviders {
		clusterProviderCVs, err := m.compGetter.GetReferencedComponentVersionsRecursive(ctx, m.compGetter.RootComponentVersion(), "cluster-provider-"+clusterProvider.Name)
		if err != nil {
			return fmt.Errorf("failed to get component version for cluster provider %s: %w", clusterProvider, err)
		}
		if len(clusterProviderCVs) != 1 {
			return fmt.Errorf("expected exactly one component version for cluster provider %s, got %d", clusterProvider, len(clusterProviderCVs))
		}
		err = m.applyCRDsForComponentVersion(ctx, &clusterProviderCVs[0], crdDirectory)
		if err != nil {
			logger.Warnf("Failed to apply CRDs for cluster provider %s: %v", clusterProvider, err)
		}
	}

	for _, serviceProvider := range m.Config.Providers.ServiceProviders {
		serviceProviderCVs, err := m.compGetter.GetReferencedComponentVersionsRecursive(ctx, m.compGetter.RootComponentVersion(), "service-provider-"+serviceProvider.Name)
		if err != nil {
			return fmt.Errorf("failed to get component version for service provider %s: %w", serviceProvider, err)
		}
		if len(serviceProviderCVs) != 1 {
			return fmt.Errorf("expected exactly one component version for service provider %s, got %d", serviceProvider, len(serviceProviderCVs))
		}
		err = m.applyCRDsForComponentVersion(ctx, &serviceProviderCVs[0], crdDirectory)
		if err != nil {
			logger.Warnf("Failed to apply CRDs for service provider %s: %v", serviceProvider, err)
		}
	}

	for _, platformService := range m.Config.Providers.PlatformServices {
		platformServiceCVs, err := m.compGetter.GetReferencedComponentVersionsRecursive(ctx, m.compGetter.RootComponentVersion(), "platform-service-"+platformService.Name)
		if err != nil {
			return fmt.Errorf("failed to get component version for platform service %s: %w", platformService, err)
		}
		if len(platformServiceCVs) != 1 {
			return fmt.Errorf("expected exactly one component version for platform service %s, got %d", platformService, len(platformServiceCVs))
		}
		err = m.applyCRDsForComponentVersion(ctx, &platformServiceCVs[0], crdDirectory)
		if err != nil {
			logger.Warnf("Failed to apply CRDs for platform service %s: %v", platformService, err)
		}
	}

	entries, err := os.ReadDir(crdDirectory)
	if err != nil {
		return fmt.Errorf("failed to read CRD download directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			fileName := entry.Name()
			if filepath.Ext(fileName) == ".yaml" || filepath.Ext(fileName) == ".yml" {
				// parse file into unstructured object
				filePath := filepath.Join(crdDirectory, fileName)
				file, err := os.Open(filePath)
				if err != nil {
					return fmt.Errorf("failed to open CRD file %s: %w", filePath, err)
				}
				defer func(path string) {
					err := file.Close()
					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "failed to close CRD file %s: %v\n", path, err)
					}
				}(filePath)

				manifestBytes, err := io.ReadAll(file)
				if err != nil {
					return fmt.Errorf("failed to read CRD file %s: %w", filePath, err)
				}

				var manifest unstructured.Unstructured
				err = yaml.Unmarshal(manifestBytes, &manifest)
				if err != nil {
					return fmt.Errorf("failed to unmarshal CRD file %s: %w", filePath, err)
				}

				if !crdIsForPlatformCluster(&manifest) {
					// if the CRD is not for the platform cluster, remove it
					logger.Tracef("Removing CRD file %s as it is not for the platform cluster", filePath)
					err = os.Remove(filePath)
					if err != nil {
						return fmt.Errorf("failed to remove CRD file %s: %w", filePath, err)
					}
				} else {
					logger.Tracef("Added CRD file: %s", filePath)
					m.crdFiles = append(m.crdFiles, filePath)
				}
			}
		}
	}

	workTree, err := m.gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	_, err = workTree.Add(filepath.Join(ResourcesDirectoryName, OpenMCPDirectoryName, CRDsDirectoryName))
	if err != nil {
		return fmt.Errorf("failed to add CRD files: %w", err)
	}

	return nil
}

func (m *DeploymentRepoManager) applyCRDsForComponentVersion(ctx context.Context, cv *ocmcli.ComponentVersion, targetDirectory string) error {
	logger := log.GetLogger()

	lastSlash := strings.LastIndex(cv.Component.Name, "/")
	if lastSlash == -1 {
		return fmt.Errorf("invalid component name: %s", cv.Component.Name)
	}

	crdResourceName := cv.Component.Name[lastSlash+1:] + "-crds"

	logger.Debugf("Applying CRDs for component %s from resource %s to directory %s", cv.Component.Name, crdResourceName, targetDirectory)

	err := m.compGetter.DownloadDirectoryResource(ctx, cv, crdResourceName, targetDirectory)
	if err != nil {
		return fmt.Errorf("failed to download CRD resource: %w", err)
	}

	return nil
}

func crdIsForPlatformCluster(crd *unstructured.Unstructured) bool {
	labels := crd.GetLabels()
	if labels == nil {
		return false
	}
	if val, ok := labels["openmcp.cloud/cluster"]; ok && val == "platform" {
		return true
	}
	return false
}

// ApplyExtraManifests copies extra manifests from the specified directory to the deployment repository and stages them for commit.
func (m *DeploymentRepoManager) ApplyExtraManifests(_ context.Context) error {
	logger := log.GetLogger()
	if len(m.ExtraManifestDir) == 0 {
		logger.Infof("No extra manifest directory specified, skipping")
		return nil
	}

	// if an extra manifest directory is specified, copy its contents to the deployment repository
	logger.Infof("Applying extra manifests from %s to deployment repository", m.ExtraManifestDir)

	entries, err := os.ReadDir(m.ExtraManifestDir)
	if err != nil {
		return fmt.Errorf("failed to read extra manifest directory: %w", err)
	}

	m.extraManifests = make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			fileName := entry.Name()
			// Check if file has .yaml or .yml extension
			if filepath.Ext(fileName) == ".yaml" || filepath.Ext(fileName) == ".yml" {
				m.extraManifests = append(m.extraManifests, fileName)
				logger.Tracef("Added extra manifest: %s", fileName)
			}
		}
	}

	return nil
}

// UpdateResourcesKustomization updates the resources kustomization file in the deployment repository to include all applied resources.
func (m *DeploymentRepoManager) UpdateResourcesKustomization() error {
	logger := log.GetLogger()
	files := make([]string, 0,
		len(m.Config.Providers.ClusterProviders)+
			len(m.Config.Providers.ServiceProviders)+
			len(m.Config.Providers.PlatformServices)+
			len(m.crdFiles)+
			len(m.extraManifests))

	for _, crdFile := range m.crdFiles {
		// get the path relative to the git repo dir
		crdFile = strings.TrimPrefix(crdFile, filepath.Join(m.gitRepoDir, ResourcesDirectoryName, OpenMCPDirectoryName)+string(os.PathSeparator))
		files = append(files, crdFile)
	}

	for _, clusterProvider := range m.Config.Providers.ClusterProviders {
		files = append(files, filepath.Join("cluster-providers", clusterProvider.Name+".yaml"))
	}

	for _, serviceProvider := range m.Config.Providers.ServiceProviders {
		files = append(files, filepath.Join("service-providers", serviceProvider.Name+".yaml"))
	}

	for _, platformService := range m.Config.Providers.PlatformServices {
		files = append(files, filepath.Join("platform-services", platformService.Name+".yaml"))
	}

	for _, manifest := range m.extraManifests {
		files = append(files, filepath.Join(ExtraManifestsDirectory, filepath.Base(manifest)))
	}

	// open resources root customization
	resourcesRootKustomizationPath := filepath.Join(ResourcesDirectoryName, OpenMCPDirectoryName, "kustomization.yaml")

	workTree, err := m.gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	fileInWorkTree, err := workTree.Filesystem.OpenFile(resourcesRootKustomizationPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s in worktree: %w", resourcesRootKustomizationPath, err)
	}

	defer func(pathInRepo billy.File) {
		err := pathInRepo.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close file in worktree: %v\n", err)
		}
	}(fileInWorkTree)

	kustomization := &KubernetesKustomization{}
	err = kustomization.ParseFromFile(fileInWorkTree)
	if err != nil {
		return fmt.Errorf("failed to parse resources root kustomization: %w", err)
	}

	_, err = fileInWorkTree.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek resources root kustomization: %w", err)
	}

	err = fileInWorkTree.Truncate(0)
	if err != nil {
		return fmt.Errorf("failed to truncate resources root kustomization: %w", err)
	}

	logger.Debugf("Adding files to resources root kustomization: %v", files)
	kustomization.AddResources(files)

	err = kustomization.WriteToFile(fileInWorkTree)
	if err != nil {
		return fmt.Errorf("failed to write resources root kustomization: %w", err)
	}

	if _, err = workTree.Add(resourcesRootKustomizationPath); err != nil {
		return fmt.Errorf("failed to add resources root kustomization to git index: %w", err)
	}

	return nil
}

// RunKustomizeAndApply runs kustomize on the environment directory and applies the resulting manifests to the target cluster.
func (m *DeploymentRepoManager) RunKustomizeAndApply(ctx context.Context) error {
	logger := log.GetLogger()
	fs := filesys.MakeFsOnDisk()
	opts := krusty.MakeDefaultOptions()
	kustomizer := krusty.MakeKustomizer(opts)

	logger.Infof("Running kustomize on %s", filepath.Join(m.gitRepoDir, EnvsDirectoryName, m.Config.Environment))
	resourceMap, err := kustomizer.Run(fs, filepath.Join(m.gitRepoDir, EnvsDirectoryName, m.Config.Environment))
	if err != nil {
		return fmt.Errorf("failed to run kustomize: %w", err)
	}
	resourcesYAML, err := resourceMap.AsYaml()
	if err != nil {
		return fmt.Errorf("failed to convert kustomized resources to YAML: %w", err)
	}

	reader := bytes.NewReader(resourcesYAML)
	manifests, err := util.ParseManifests(reader)
	if err != nil {
		return fmt.Errorf("failed to parse kustomized resources: %w", err)
	}

	for _, manifest := range manifests {
		if manifest.GetKind() == "Kustomization" && strings.Contains(manifest.GetAPIVersion(), "kustomize.toolkit.fluxcd.io") {
			logger.Infof("Applying Kustomization manifest: %s/%s", manifest.GetNamespace(), manifest.GetName())
			err = util.CreateOrUpdate(ctx, m.TargetCluster, manifest)
			if err != nil {
				return fmt.Errorf("failed to apply Kustomization manifest %s/%s: %w", manifest.GetNamespace(), manifest.GetName(), err)
			}
		}
	}

	return nil
}

// CommitAndPushChanges commits all changes in the deployment repository and pushes them to the remote repository.
// If there are no changes to commit, it does nothing.
func (m *DeploymentRepoManager) CommitAndPushChanges(_ context.Context) error {
	logger := log.GetLogger()

	logger.Info("Committing and pushing changes to deployment repository")

	err := CommitChanges(m.gitRepo, "apply templates", "openmcp", "noreply@openmcp.cloud")
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	err = PushRepo(m.gitRepo, m.Config.DeploymentRepository.RepoBranch, m.gitConfig)
	if err != nil {
		return fmt.Errorf("failed to push changes to deployment repository: %w", err)
	}

	return nil
}

// GitRepoDir returns the path to the cloned deployment repository.
func (m *DeploymentRepoManager) GitRepoDir() string {
	return m.gitRepoDir
}
