package deploymentrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/openmcp-project/bootstrapper/internal/config"

	gitconfig "github.com/openmcp-project/bootstrapper/internal/git-config"
	"github.com/openmcp-project/bootstrapper/internal/log"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

const (
	OpenMCPOperatorComponentName              = "openmcp-operator"
	FluxCDSourceControllerResourceName        = "fluxcd-source-controller"
	FluxCDKustomizationControllerResourceName = "fluxcd-kustomize-controller"
	FluxCDHelmControllerResourceName          = "fluxcd-helm-controller"

	EnvsDirectoryName      = "envs"
	ResourcesDirectoryName = "resources"
	OpenMCPDirectoryName   = "openmcp"
	FluxCDDirectoryName    = "fluxcd"
	CRDsDirectoryName      = "crds"
)

// DeploymentRepoManager manages the deployment repository by applying templates and committing changes.
type DeploymentRepoManager struct {
	// Vars set via constructor or "With" methods

	// GitConfigPath is the path to the Git configuration file
	GitConfigPath string
	// OcmConfigPath is the path to the OCM configuration file
	// +optional
	OcmConfigPath string

	Config *config.BootstrapperConfig

	// TargetCluster is the Kubernetes cluster to which the deployment will be applied
	TargetCluster *clusters.Cluster

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
}

// NewDeploymentRepoManager creates a new DeploymentRepoManager with the specified parameters.
func NewDeploymentRepoManager(config *config.BootstrapperConfig, targetCluster *clusters.Cluster, gitConfigPath, ocmConfigPath string) *DeploymentRepoManager {
	return &DeploymentRepoManager{
		Config:        config,
		TargetCluster: targetCluster,
		GitConfigPath: gitConfigPath,
		OcmConfigPath: ocmConfigPath,
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

	m.openMCPOperatorCV, err = m.compGetter.GetReferencedComponentVersionRecursive(ctx, m.compGetter.RootComponentVersion(), OpenMCPOperatorComponentName)
	if err != nil {
		return m, fmt.Errorf("failed to get openmcp-operator component version: %w", err)
	}

	m.fluxcdCV, err = m.compGetter.GetComponentVersionForResourceRecursive(ctx, m.compGetter.RootComponentVersion(), FluxCDSourceControllerResourceName)
	if err != nil {
		return m, fmt.Errorf("failed to get fluxcd source controller component version: %w", err)
	}

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

	logger.Infof("Applying templates from %s/%s to deployment repository", m.Config.Component.FluxcdTemplateResourcePath, m.Config.Component.OpenMCPOperatorTemplateResourcePath)
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

	templateInput["fluxCDEnvPath"] = "./" + EnvsDirectoryName + "/" + m.Config.Environment + "/" + FluxCDDirectoryName
	templateInput["gitRepoEnvBranch"] = m.Config.DeploymentRepository.RepoBranch
	templateInput["fluxCDResourcesPath"] = "../../../" + ResourcesDirectoryName + "/" + FluxCDDirectoryName
	templateInput["openMCPResourcesPath"] = "../../../" + ResourcesDirectoryName + "/" + OpenMCPDirectoryName
	templateInput["git"] = map[string]interface{}{
		"repoUrl":    m.Config.DeploymentRepository.RepoURL,
		"mainBranch": m.Config.DeploymentRepository.RepoBranch,
	}

	templateInput["images"] = make(map[string]interface{})

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

	err = TemplateDir(m.templatesDir, templateInput, m.gitRepo)
	if err != nil {
		return fmt.Errorf("failed to apply templates from directory %s: %w", m.templatesDir, err)
	}

	/*
		workTree, err := m.gitRepo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		workTreePath := filepath.Join(ResourcesDirectoryName, OpenMCPDirectoryName, "extra")

			for _, manifest := range m.Config.OpenMCPOperator.Manifests {
				workTreeFile := filepath.Join(workTreePath, manifest.Name+".yaml")
				logger.Infof("Applying openmcp-operator manifest %s to deployment repository", manifest.Name)

				manifestRaw, err := yaml.Marshal(manifest.ManifestParsed)
				if err != nil {
					return fmt.Errorf("failed to marshal openmcp-operator manifest %s: %w", manifest.Name, err)
				}

				err = os.MkdirAll(filepath.Join(m.gitRepoDir, workTreePath), 0755)
				if err != nil {
					return fmt.Errorf("failed to create directory %s in deployment repository: %w", workTreePath, err)
				}

				err = os.WriteFile(filepath.Join(m.gitRepoDir, workTreeFile), manifestRaw, 0o644)
				if err != nil {
					return fmt.Errorf("failed to write openmcp-operator manifest %s to deployment repository: %w", manifest.Name, err)
				}
				_, err = workTree.Add(workTreePath)
				if err != nil {
					return fmt.Errorf("failed to add openmcp-operator manifest %s to git index: %w", manifest.Name, err)
				}
			}
	*/

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

	logger.Infof("Applying Custom Resource Definitions to deployment repository")

	crdsDownloadDir := filepath.Join(m.gitRepoDir, ResourcesDirectoryName, OpenMCPDirectoryName, CRDsDirectoryName)

	// if the CRDs directory already exists, remove it to ensure a clean state
	if _, err := os.Stat(crdsDownloadDir); err == nil {
		err = os.RemoveAll(crdsDownloadDir)
		if err != nil {
			return fmt.Errorf("failed to remove existing CRD directory: %w", err)
		}
	}

	err := os.Mkdir(crdsDownloadDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create CRD download directory: %w", err)
	}

	err = m.compGetter.DownloadDirectoryResource(ctx, m.openMCPOperatorCV, "openmcp-operator-crds", crdsDownloadDir)
	if err != nil {
		return fmt.Errorf("failed to download CRD resource: %w", err)
	}

	// List all YAML files in the CRDs download directory
	entries, err := os.ReadDir(crdsDownloadDir)
	if err != nil {
		return fmt.Errorf("failed to read CRD download directory: %w", err)
	}

	m.crdFiles = make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			fileName := entry.Name()
			// Check if file has .yaml or .yml extension
			if filepath.Ext(fileName) == ".yaml" || filepath.Ext(fileName) == ".yml" {
				filePath := filepath.Join(crdsDownloadDir, fileName)
				m.crdFiles = append(m.crdFiles, filePath)
				logger.Tracef("Added CRD file: %s", filePath)
			}
		}
	}

	logger.Infof("Found %d CRD files", len(m.crdFiles))

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

func (m *DeploymentRepoManager) UpdateResourcesKustomization() error {
	logger := log.GetLogger()
	files := make([]string, 0,
		len(m.Config.Providers.ClusterProviders)+
			len(m.Config.Providers.ServiceProviders)+
			len(m.Config.Providers.PlatformServices)+
			len(m.crdFiles))

	// len(m.Config.OpenMCPOperator.Manifests))

	for _, crdFile := range m.crdFiles {
		files = append(files, filepath.Join(CRDsDirectoryName, filepath.Base(crdFile)))
	}

	for _, clusterProvider := range m.Config.Providers.ClusterProviders {
		files = append(files, filepath.Join("cluster-providers", clusterProvider+".yaml"))
	}

	for _, serviceProvider := range m.Config.Providers.ServiceProviders {
		files = append(files, filepath.Join("service-providers", serviceProvider+".yaml"))
	}

	for _, platformService := range m.Config.Providers.PlatformServices {
		files = append(files, filepath.Join("platform-services", platformService+".yaml"))
	}

	/*
		for _, manifest := range m.Config.OpenMCPOperator.Manifests {
			files = append(files, filepath.Join("extra", manifest.Name+".yaml"))
		}
	*/

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

	logger.Infof("Applying kustomized resources to target cluster")
	err = util.ApplyManifests(ctx, m.TargetCluster, resourcesYAML)
	if err != nil {
		return fmt.Errorf("failed to apply kustomized resources to cluster: %w", err)
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

func (m *DeploymentRepoManager) GitRepoDir() string {
	return m.gitRepoDir
}
