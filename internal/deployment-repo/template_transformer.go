package deploymentrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	fluxk "github.com/fluxcd/kustomize-controller/api/v1"
	fluxm "github.com/fluxcd/pkg/apis/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"

	"github.com/openmcp-project/bootstrapper/internal/log"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

const (
	TemplatesDirectoryName = "templates"
)

type TemplateTransformer struct {
	ComponentGetter                  *ocmcli.ComponentGetter
	FluxTemplateResourceLocation     string
	OpenMMCPTemplateResourceLocation string
	InitializeRESTConfig             string
	WorkDir                          string
}

func NewTemplateTransformer(componentGetter *ocmcli.ComponentGetter, fluxTemplateResourceLocation, openMMCPTemplateResourceLocation, workDir string) *TemplateTransformer {
	return &TemplateTransformer{
		ComponentGetter:                  componentGetter,
		FluxTemplateResourceLocation:     fluxTemplateResourceLocation,
		OpenMMCPTemplateResourceLocation: openMMCPTemplateResourceLocation,
		WorkDir:                          workDir,
	}
}

func (t *TemplateTransformer) Transform(ctx context.Context, envName, targetDir string) error {
	logger := log.GetLogger()

	downloadDir := filepath.Join(t.WorkDir, "transformer", "download")
	logger.Tracef("Using download directory: %s", downloadDir)
	err := os.MkdirAll(downloadDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	// clean the targetDir if it already exists
	err = os.RemoveAll(targetDir)
	if err != nil {
		return fmt.Errorf("failed to clean target directory: %w", err)
	}

	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	//// Download template resources
	logger.Infof("Downloading template resources")

	// download the fluxcd template resource to <downloadDir>/fluxcd
	fluxcdDownloadDir := filepath.Join(downloadDir, FluxCDDirectoryName)
	logger.Debugf("Downloading fluxcd template resource to: %s", fluxcdDownloadDir)
	err = t.ComponentGetter.DownloadDirectoryResourceByLocation(ctx, t.ComponentGetter.RootComponentVersion(), t.FluxTemplateResourceLocation, fluxcdDownloadDir)
	if err != nil {
		return fmt.Errorf("failed to download fluxcd template resource: %w", err)
	}

	// download the openmcp template resource to <downloadDir>/openmcp
	openMMCPDownloadDir := filepath.Join(downloadDir, OpenMCPDirectoryName)
	logger.Debugf("Downloading openmmcp template resource to: %s", openMMCPDownloadDir)
	err = t.ComponentGetter.DownloadDirectoryResourceByLocation(ctx, t.ComponentGetter.RootComponentVersion(), t.OpenMMCPTemplateResourceLocation, openMMCPDownloadDir)
	if err != nil {
		return fmt.Errorf("failed to download openmmcp template resource: %w", err)
	}

	//// Create directory structure
	logger.Info("Transforming templates into deployment repository structure")

	// create directory <targetDir>/envs/<envName>/fluxcd and <targetDir>/envs/<envName>/openmmcp
	fluxCDEnvDir := filepath.Join(targetDir, EnvsDirectoryName, envName, FluxCDDirectoryName)
	err = os.MkdirAll(fluxCDEnvDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create fluxcd environment directory: %w", err)
	}

	openMMCPEnvDir := filepath.Join(targetDir, EnvsDirectoryName, envName, OpenMCPDirectoryName)
	err = os.MkdirAll(openMMCPEnvDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create openmmcp environment directory: %w", err)
	}

	// create directory <targetDir>/resources/fluxcd and <targetDir>/resources/openmcp
	fluxCDResourcesDir := filepath.Join(targetDir, ResourcesDirectoryName, FluxCDDirectoryName)
	err = os.MkdirAll(fluxCDResourcesDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create fluxcd resources directory: %w", err)
	}

	openMMCPResourcesDir := filepath.Join(targetDir, ResourcesDirectoryName, OpenMCPDirectoryName)
	err = os.MkdirAll(openMMCPResourcesDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create openmmcp resources directory: %w", err)
	}

	//// Copy files from downloaded templates to target directories
	logger.Debug("Copying template files to target directories")

	// copy all files from <fluxDownloadDir>/templates/overlays to <targetDir>/envs/<envName>/fluxcd
	err = util.CopyDir(filepath.Join(fluxcdDownloadDir, TemplatesDirectoryName, "overlays"), fluxCDEnvDir)
	if err != nil {
		return fmt.Errorf("failed to copy fluxcd overlays: %w", err)
	}

	// copy all files from <fluxDownloadDir>/templates/resources to <targetDir>/resources/fluxcd
	err = util.CopyDir(filepath.Join(fluxcdDownloadDir, TemplatesDirectoryName, ResourcesDirectoryName), fluxCDResourcesDir)
	if err != nil {
		return fmt.Errorf("failed to copy fluxcd resources: %w", err)
	}

	// copy all files from <openMMCPDownloadDir>/templates/overlays to <targetDir>/envs/<envName>/openmmcp
	err = util.CopyDir(filepath.Join(openMMCPDownloadDir, TemplatesDirectoryName, "overlays"), openMMCPEnvDir)
	if err != nil {
		return fmt.Errorf("failed to copy openmmcp overlays: %w", err)
	}

	// copy all files from <openMMCPDownloadDir>/templates/resources to <targetDir>/resources/openmmcp
	err = util.CopyDir(filepath.Join(openMMCPDownloadDir, TemplatesDirectoryName, ResourcesDirectoryName), openMMCPResourcesDir)
	if err != nil {
		return fmt.Errorf("failed to copy openmmcp resources: %w", err)
	}

	// create the <targetDir>/envs/<envName>/kustomization.yaml file
	kustomizationDir := filepath.Join(targetDir, EnvsDirectoryName, envName)
	kustomizationFile := filepath.Join(kustomizationDir, "kustomization.yaml")
	logger.Debugf("Creating Kubernetes kustomization file %s", kustomizationFile)
	err = writeKubernetesKustomization([]string{"../../" + ResourcesDirectoryName, OpenMCPDirectoryName}, []string{"root-kustomization.yaml"}, "kustomization", kustomizationDir)
	if err != nil {
		return fmt.Errorf("failed to write environment kustomization file: %w", err)
	}

	// create the <targetDir>/envs/<envName>/root-kustomization.yaml file
	kustomizationDir = filepath.Join(targetDir, EnvsDirectoryName, envName)
	kustomizationFile = filepath.Join(kustomizationDir, "root-kustomization.yaml")
	logger.Debugf("Creating Flux kustomization patch file %s", kustomizationFile)
	rootKustomization := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "bootstrap",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"path": "./" + EnvsDirectoryName + "/" + envName,
		},
	}
	err = writeFluxKustomizationPatch(rootKustomization, "root-kustomization", kustomizationDir)
	if err != nil {
		return fmt.Errorf("failed to write root kustomization file: %w", err)
	}

	// create the <targetDir>/resources/kustomization.yaml file
	logger.Debugf("Creating Kubernetes kustomization file %s", filepath.Join(targetDir, ResourcesDirectoryName, "kustomization.yaml"))
	err = writeKubernetesKustomization([]string{"root-kustomization.yaml"}, nil, "kustomization", filepath.Join(targetDir, ResourcesDirectoryName))
	if err != nil {
		return fmt.Errorf("failed to write resources kustomization file: %w", err)
	}

	// create the <targetDir>/resources/root-kustomization.yaml file
	kustomizationDir = filepath.Join(targetDir, ResourcesDirectoryName)
	kustomizationFile = filepath.Join(kustomizationDir, "root-kustomization.yaml")
	logger.Debugf("Creating Flux kustomization file %s", kustomizationFile)
	rootResourcesKustomization := &fluxk.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bootstrap",
			Namespace: "default",
		},
		Spec: fluxk.KustomizationSpec{
			Interval: metav1.Duration{Duration: 10 * time.Minute},
			Path:     "<templated>",
			Prune:    true,
			SourceRef: fluxk.CrossNamespaceSourceReference{
				Kind:      "GitRepository",
				Name:      "environments",
				Namespace: "flux-system",
			},
			DependsOn: []fluxm.NamespacedObjectReference{
				{
					Name:      "flux-system",
					Namespace: "flux-system",
				},
			},
		},
	}
	err = writeFluxKustomization(rootResourcesKustomization, "root-kustomization", kustomizationDir)
	if err != nil {
		return fmt.Errorf("failed to write root resources kustomization file: %w", err)
	}

	return nil
}

func writeKubernetesKustomization(resources, patches []string, name, targetDir string) error {
	k := &KubernetesKustomization{
		Kustomization: ktypes.Kustomization{
			TypeMeta: ktypes.TypeMeta{
				APIVersion: ktypes.KustomizationVersion,
				Kind:       ktypes.KustomizationKind,
			},
			Resources: resources,
			Patches:   make([]ktypes.Patch, 0, len(patches)),
		},
	}

	for _, p := range patches {
		k.Kustomization.Patches = append(k.Kustomization.Patches, ktypes.Patch{
			Path: p,
		})
	}

	file, err := os.Create(filepath.Join(targetDir, name+".yaml"))
	if err != nil {
		return fmt.Errorf("failed to create kustomization file: %w", err)
	}

	return k.WriteToFile(file)
}

func writeFluxKustomization(kustomization *fluxk.Kustomization, name, targetDir string) error {
	k := &FluxKustomization{
		Kustomization: *kustomization,
	}

	k.TypeMeta = metav1.TypeMeta{
		APIVersion: fluxk.GroupVersion.String(),
		Kind:       "Kustomization",
	}

	file, err := os.Create(filepath.Join(targetDir, name+".yaml"))
	if err != nil {
		return fmt.Errorf("failed to create kustomization file: %w", err)
	}

	return k.WriteToFile(file)
}

func writeFluxKustomizationPatch(kustomizationPatch map[string]interface{}, name, targetDir string) error {
	kustomizationPatch["apiVersion"] = fluxk.GroupVersion.String()
	kustomizationPatch["kind"] = "Kustomization"

	patchRaw, err := yaml.Marshal(kustomizationPatch)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization patch: %w", err)
	}

	patchPath := filepath.Join(targetDir, name+".yaml")
	err = os.WriteFile(patchPath, patchRaw, 0644)
	if err != nil {
		return fmt.Errorf("failed to write kustomization patch file: %w", err)
	}
	return nil
}
