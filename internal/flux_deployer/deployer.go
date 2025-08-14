package flux_deployer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/template"
)

type FluxDeployer struct {
	componentLocation string

	// deploymentTemplates is the path to the deployment templates in the templates component.
	// It consists of segments separated by slashes. All segments except the last one are names of component references.
	// The last segment is the name of a resource in the component, which is reached by starting at the root component and going through the component references.
	// For example, "gitops-templates/fluxcd" means: start at the root component, go to the component reference "gitops-templates", and then use the resource named "fluxcd" in that component.
	deploymentTemplates        string
	deploymentRepository       string
	deploymentRepositoryBranch string
	deploymentRepositoryPath   string
	ocmConfig                  string
	gitCredentials             string
	fluxcdNamespace            string
	platformKubeconfig         string
	platformCluster            *clusters.Cluster
	log                        *logrus.Logger
}

func NewFluxDeployer(componentLocation, deploymentTemplates, deploymentRepository, deploymentRepositoryBranch, deploymentRepositoryPath,
	ocmConfig, gitCredentials, fluxcdNamespace, platformKubeconfig string, platformCluster *clusters.Cluster, log *logrus.Logger) *FluxDeployer {

	return &FluxDeployer{
		componentLocation:          componentLocation,
		deploymentTemplates:        deploymentTemplates,
		deploymentRepository:       deploymentRepository,
		deploymentRepositoryBranch: deploymentRepositoryBranch,
		deploymentRepositoryPath:   deploymentRepositoryPath,
		ocmConfig:                  ocmConfig,
		gitCredentials:             gitCredentials,
		fluxcdNamespace:            fluxcdNamespace,
		platformKubeconfig:         platformKubeconfig,
		platformCluster:            platformCluster,
		log:                        log,
	}
}

func (d *FluxDeployer) Deploy(ctx context.Context) error {

	// Get root component and gitops-templates component.
	d.log.Info("Loading root component and gitops-templates component")
	componentGetter := ocmcli.NewComponentGetter(d.componentLocation, d.deploymentTemplates, d.ocmConfig)
	if err := componentGetter.InitializeComponents(ctx); err != nil {
		return err
	}

	// Create a temporary directory to store the downloaded resource
	d.log.Info("Creating download directory for gitops-templates")
	downloadDir, err := os.MkdirTemp("", "flux-resource-")
	if err != nil {
		return fmt.Errorf("error creating temporary download directory for flux resource: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(downloadDir); err != nil {
			fmt.Printf("error removing temporary download directory for flux resource: %v\n", err)
		}
	}()
	d.log.Debugf("Download directory: %s", downloadDir)

	// Download resource from gitops-templates component into the download directory
	d.log.Info("Downloading gitops-templates")
	if err := componentGetter.DownloadTemplatesResource(ctx, downloadDir); err != nil {
		return fmt.Errorf("error downloading templates: %w", err)
	}

	if err := d.DeployFluxControllers(ctx, componentGetter.RootComponentVersion(), downloadDir); err != nil {
		return fmt.Errorf("error deploying flux controllers: %w", err)
	}

	if err := d.establishFluxSync(ctx, downloadDir); err != nil {
		return fmt.Errorf("error establishing flux synchronization: %w", err)
	}

	return nil
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
			"namespace": d.fluxcdNamespace,
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
	if err := d.applyManifests(ctx, manifest); err != nil {
		return err
	}

	return nil
}

func (d *FluxDeployer) establishFluxSync(ctx context.Context, downloadDir string) error {
	d.log.Info("Establishing flux synchronization with deployment repository")

	const secretName = "git"

	if err := CreateGitCredentialsSecret(ctx, d.log, d.gitCredentials, secretName, d.fluxcdNamespace, d.platformCluster.Client()); err != nil {
		return err
	}

	// Read manifest file
	filepath := path.Join(downloadDir, "resources", "gotk-sync.yaml")
	d.log.Debugf("Reading flux synchronization objects from file %s", filepath)
	manifestTpl, err := d.readFileContent(filepath)
	if err != nil {
		return fmt.Errorf("error reading manifests for flux sync: %w", err)
	}

	// Template
	d.log.Debug("Templating flux synchronization objects")
	values := map[string]any{
		"Values": map[string]any{
			"namespace": d.fluxcdNamespace,
			"git": map[string]any{
				"repoUrl":    d.deploymentRepository,
				"mainBranch": d.deploymentRepositoryBranch,
				"path":       d.deploymentRepositoryPath,
				"secretName": secretName,
			},
		},
	}
	manifest, err := template.NewTemplateExecution().Execute("flux-sync", string(manifestTpl), values)
	if err != nil {
		return fmt.Errorf("error templating flux synchronization objects: %w", err)
	}

	// Apply
	d.log.Debug("Applying flux synchronization objects")
	if err := d.applyManifests(ctx, manifest); err != nil {
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

func (d *FluxDeployer) applyManifests(ctx context.Context, manifests []byte) error {

	// Parse manifests into unstructured objects
	reader := bytes.NewReader(manifests)
	unstructuredObjects, err := d.parseManifests(reader)
	if err != nil {
		return fmt.Errorf("error parsing manifests: %w", err)
	}

	// Apply objects to the platform cluster
	for _, u := range unstructuredObjects {
		if err := d.applyUnstructuredObject(ctx, u); err != nil {
			return err
		}
	}

	return nil
}

func (d *FluxDeployer) parseManifests(reader io.Reader) ([]*unstructured.Unstructured, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	var result []*unstructured.Unstructured
	for {
		u := &unstructured.Unstructured{}
		err := decoder.Decode(u)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		if len(u.Object) == 0 {
			continue
		}
		result = append(result, u)
	}
	return result, nil
}

func (d *FluxDeployer) applyUnstructuredObject(ctx context.Context, u *unstructured.Unstructured) error {
	objectKey := client.ObjectKeyFromObject(u)
	objectLogString := fmt.Sprintf("%s %s", u.GetObjectKind(), objectKey.String())
	fmt.Printf("Applying object %s\n", objectLogString)

	existingObj := &unstructured.Unstructured{}
	existingObj.SetGroupVersionKind(u.GroupVersionKind())
	getErr := d.platformCluster.Client().Get(ctx, objectKey, existingObj)
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// create object
			createErr := d.platformCluster.Client().Create(ctx, u)
			if createErr != nil {
				return fmt.Errorf("error creating object %s: %w", objectLogString, createErr)
			}
		} else {
			return fmt.Errorf("error reading object %s: %w", objectLogString, getErr)
		}
	} else {
		// update object
		u.SetResourceVersion(existingObj.GetResourceVersion())
		updateErr := d.platformCluster.Client().Update(ctx, u)
		if updateErr != nil {
			return fmt.Errorf("error updating object %s: %w", objectLogString, updateErr)
		}
	}
	return nil
}
