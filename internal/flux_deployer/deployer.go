package flux_deployer

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	cfg "github.com/openmcp-project/bootstrapper/internal/config"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/template"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

const (
	EnvsDirectoryName      = "envs"
	FluxCDDirectoryName    = "fluxcd"
	ResourcesDirectoryName = "resources"
	TemplatesDirectoryName = "templates"
	OverlaysDirectoryName  = "overlays"

	FluxCDSourceControllerResourceName        = "fluxcd-source-controller"
	FluxCDKustomizationControllerResourceName = "fluxcd-kustomize-controller"
	FluxCDHelmControllerResourceName          = "fluxcd-helm-controller"
)

type FluxDeployer struct {
	Config *cfg.BootstrapperConfig

	// GitConfigPath is the path to the Git configuration file
	GitConfigPath string
	// OcmConfigPath is the path to the OCM configuration file
	OcmConfigPath string

	platformCluster *clusters.Cluster
	fluxNamespace   string
	// fluxcdCV is the component version of the fluxcd source controller component
	fluxcdCV *ocmcli.ComponentVersion
	log      *logrus.Logger

	workDir      string
	downloadDir  string
	templatesDir string
	repoDir      string
}

func NewFluxDeployer(config *cfg.BootstrapperConfig, gitConfigPath, ocmConfigPath string, platformCluster *clusters.Cluster, log *logrus.Logger) *FluxDeployer {
	return &FluxDeployer{
		Config:          config,
		GitConfigPath:   gitConfigPath,
		OcmConfigPath:   ocmConfigPath,
		platformCluster: platformCluster,
		fluxNamespace:   FluxSystemNamespace,
		log:             log,
	}
}

func (d *FluxDeployer) Deploy(ctx context.Context) (err error) {

	if err := CreateGitCredentialsSecret(ctx, d.log, d.GitConfigPath, GitSecretName, d.fluxNamespace, d.platformCluster.Client()); err != nil {
		return err
	}

	// Create temporary working directory
	d.log.Info("Creating working directory for gitops-templates")
	d.workDir, err = util.CreateTempDir()
	if err != nil {
		return fmt.Errorf("error creating temporary working directory for flux resource: %w", err)
	}
	defer func() {
		err := util.DeleteTempDir(d.workDir)
		if err != nil {
			fmt.Printf("error removing temporary working directory for flux resource: %v\n", err)
		}
	}()
	d.log.Tracef("Created working directory: %s", d.workDir)

	d.downloadDir = filepath.Join(d.workDir, "download")
	d.log.Tracef("Creating download directory: %s", d.downloadDir)
	err = os.MkdirAll(d.downloadDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}
	d.log.Tracef("Created download directory: %s", d.downloadDir)

	d.templatesDir = filepath.Join(d.workDir, "templates")
	d.log.Tracef("Creating templates directory: %s", d.templatesDir)
	err = os.MkdirAll(d.templatesDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}
	d.log.Tracef("Created templates directory: %s", d.templatesDir)

	d.repoDir = filepath.Join(d.workDir, "repo")
	d.log.Tracef("Creating repo directory: %s", d.repoDir)
	err = os.MkdirAll(d.repoDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}
	d.log.Tracef("Created repo directory: %s", d.repoDir)

	// Get components
	// - root component
	// - gitops-templates component
	// - that contains the image resources for fluxcd source controller component
	d.log.Info("Loading root component and gitops-templates component")
	componentGetter := ocmcli.NewComponentGetter(d.Config.Component.OpenMCPComponentLocation, d.Config.Component.FluxcdTemplateResourcePath, d.OcmConfigPath)
	if err := componentGetter.InitializeComponents(ctx); err != nil {
		return err
	}

	d.fluxcdCV, err = componentGetter.GetComponentVersionForResourceRecursive(ctx, componentGetter.RootComponentVersion(), FluxCDSourceControllerResourceName)
	if err != nil {
		return fmt.Errorf("failed to get fluxcd source controller component version: %w", err)
	}

	// Download resource from gitops-templates component into the download directory
	d.log.Info("Downloading gitops-templates")
	if err := componentGetter.DownloadTemplatesResource(ctx, d.downloadDir); err != nil {
		return fmt.Errorf("error downloading templates: %w", err)
	}

	// Copy files from <workdir>/download to <workdir>/templates, re-arranging the directory structure as needed for kustomize
	if err := d.ArrangeTemplates(); err != nil {
		return fmt.Errorf("error arranging templates directory: %w", err)
	}

	// Template all files in <workdir>/templates, and write the result to <workdir>/repo
	if err := d.Template(); err != nil {
		return fmt.Errorf("error templating files: %w", err)
	}

	// Kustomize <workdir>/repo/envs/<envName>/fluxcd
	fluxCDEnvDir := filepath.Join(d.repoDir, EnvsDirectoryName, d.Config.Environment, FluxCDDirectoryName)
	manifests, err := d.Kustomize(fluxCDEnvDir)
	if err != nil {
		return fmt.Errorf("error kustomizing templated files: %w", err)
	}

	// Apply manifests to the platform cluster
	d.log.Info("Applying flux deployment objects")
	if err := util.ApplyManifests(ctx, d.platformCluster, manifests); err != nil {
		return err
	}

	return nil
}

// ArrangeTemplates fills the templates directory with the files from the download directory, adjusting the directory structure as needed for the kustomization.
func (d *FluxDeployer) ArrangeTemplates() (err error) {
	d.log.Info("Arranging template files")

	// Create directory <templatesDir>/envs/<envName>/fluxcd
	fluxCDEnvDir := filepath.Join(d.templatesDir, EnvsDirectoryName, d.Config.Environment, FluxCDDirectoryName)
	err = os.MkdirAll(fluxCDEnvDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create fluxcd environment directory: %w", err)
	}

	// Create directory <templatesDir>/resources/fluxcd
	fluxCDResourcesDir := filepath.Join(d.templatesDir, ResourcesDirectoryName, FluxCDDirectoryName)
	err = os.MkdirAll(fluxCDResourcesDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create fluxcd resources directory: %w", err)
	}

	d.log.Debug("Copying template files to target directories")

	// copy all files from <downloadDir>/templates/overlays to <templatesDir>/envs/<envName>/fluxcd
	err = util.CopyDir(filepath.Join(d.downloadDir, TemplatesDirectoryName, OverlaysDirectoryName), fluxCDEnvDir)
	if err != nil {
		return fmt.Errorf("failed to copy fluxcd overlays: %w", err)
	}

	// copy all files from <downloadDir>/templates/resources to <templatesDir>/resources/fluxcd
	err = util.CopyDir(filepath.Join(d.downloadDir, TemplatesDirectoryName, ResourcesDirectoryName), fluxCDResourcesDir)
	if err != nil {
		return fmt.Errorf("failed to copy fluxcd resources: %w", err)
	}

	d.log.Info("Arranged template files")
	return nil
}

// Template templates the files in the templates directory, replacing placeholders with actual values.
// The resulting files are written to the repo directory.
func (d *FluxDeployer) Template() (err error) {
	d.log.Infof("Applying templates from %s to deployment repository", d.Config.Component.FluxcdTemplateResourcePath)
	templateInput := template.NewTemplateInput()

	templateInput.SetImagePullSecrets(d.Config.ImagePullSecrets)

	templateInput["fluxCDEnvPath"] = "./" + EnvsDirectoryName + "/" + d.Config.Environment + "/" + FluxCDDirectoryName
	templateInput["gitRepoEnvBranch"] = d.Config.DeploymentRepository.RepoBranch
	templateInput["fluxCDResourcesPath"] = "../../../" + ResourcesDirectoryName + "/" + FluxCDDirectoryName

	templateInput.SetGitRepo(d.Config.DeploymentRepository)

	if err = templateInput.AddImageResource(d.fluxcdCV, FluxCDSourceControllerResourceName, "sourceController"); err != nil {
		return fmt.Errorf("failed to apply fluxcd source controller template input: %w", err)
	}
	if err = templateInput.AddImageResource(d.fluxcdCV, FluxCDKustomizationControllerResourceName, "kustomizeController"); err != nil {
		return fmt.Errorf("failed to apply fluxcd kustomize controller template input: %w", err)
	}
	if err = templateInput.AddImageResource(d.fluxcdCV, FluxCDHelmControllerResourceName, "helmController"); err != nil {
		return fmt.Errorf("failed to apply fluxcd helm controller template input: %w", err)
	}

	if err = TemplateDirectory(d.templatesDir, templateInput, d.repoDir, d.log); err != nil {
		return fmt.Errorf("failed to apply templates from directory %s: %w", d.templatesDir, err)
	}

	return nil
}

// Kustomize runs kustomize on the given directory and returns the resulting yaml as a byte slice.
func (d *FluxDeployer) Kustomize(dir string) ([]byte, error) {
	d.log.Infof("Kustomizing files in directory: %s", dir)
	fs := filesys.MakeFsOnDisk()
	opts := krusty.MakeDefaultOptions()
	kustomizer := krusty.MakeKustomizer(opts)

	resourceMap, err := kustomizer.Run(fs, dir)
	if err != nil {
		return nil, fmt.Errorf("error running kustomization: %w", err)
	}

	resourcesYaml, err := resourceMap.AsYaml()
	if err != nil {
		return nil, fmt.Errorf("error converting resources to yaml: %w", err)
	}

	return resourcesYaml, nil
}

func (d *FluxDeployer) DeployFluxControllers(ctx context.Context, rootComponentVersion *ocmcli.ComponentVersion, downloadDir string) error {
	d.log.Info("Deploying flux")

	images, err := GetFluxCDImages(rootComponentVersion)
	if err != nil {
		return fmt.Errorf("error getting images for flux controllers: %w", err)
	}

	// Read manifest file
	filepath := path.Join(downloadDir, "resources", "gotk-components.yaml")
	d.log.Debugf("Reading flux deployment objects from file %s", filepath)
	manifestTpl, err := d.readFileContent(filepath)
	if err != nil {
		return fmt.Errorf("error reading flux deployment objects from file %s: %w", filepath, err)
	}

	// Template
	values := map[string]any{
		"Values": map[string]any{
			"namespace": d.fluxNamespace,
			"images":    images,
		},
	}
	d.log.Debug("Templating flux deployment objects")
	manifest, err := template.NewTemplateExecution().Execute("flux-deployment", string(manifestTpl), values)
	if err != nil {
		return fmt.Errorf("error templating flux deployment objects: %w", err)
	}

	// Apply
	d.log.Debug("Applying flux deployment objects")
	if err := util.ApplyManifests(ctx, d.platformCluster, manifest); err != nil {
		return err
	}

	return nil
}

func (d *FluxDeployer) readFileContent(filepath string) ([]byte, error) {
	d.log.Debugf("Reading file: %s", filepath)

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist at path: %s", filepath)
	}

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filepath, err)
	}

	return content, nil
}
