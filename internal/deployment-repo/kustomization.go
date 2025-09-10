package deploymentrepo

import (
	"context"
	"fmt"
	"io"

	fluxk "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	ktypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"

	"github.com/openmcp-project/bootstrapper/internal/util"
)

type KubernetesKustomization struct {
	ktypes.Kustomization `yaml:",inline"`
}

func (k *KubernetesKustomization) ParseFromFile(file io.Reader) error {
	kustomizationRaw, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read kustomization file: %w", err)
	}

	err = yaml.Unmarshal(kustomizationRaw, k)
	if err != nil {
		return fmt.Errorf("failed to unmarshal kustomization file: %w", err)
	}

	return nil
}

func (k *KubernetesKustomization) WriteToFile(file io.Writer) error {
	kustomizationRaw, err := yaml.Marshal(k)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization file: %w", err)
	}

	_, err = file.Write(kustomizationRaw)
	if err != nil {
		return fmt.Errorf("failed to write kustomization file: %w", err)
	}

	return nil
}

func (k *KubernetesKustomization) AddResource(resource string) {
	if len(k.Resources) == 0 {
		k.Resources = make([]string, 1)
	}
	k.Resources = append(k.Resources, resource)
}

func (k *KubernetesKustomization) AddResources(resources []string) {
	if len(k.Resources) == 0 {
		k.Resources = make([]string, 0, len(resources))
	}
	k.Resources = append(k.Resources, resources...)
}

type FluxKustomization struct {
	fluxk.Kustomization `yaml:",inline"`
}

func (k *FluxKustomization) ParseFromFile(file io.Reader) error {
	kustomizationRaw, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read kustomization file: %w", err)
	}

	err = yaml.Unmarshal(kustomizationRaw, k)
	if err != nil {
		return fmt.Errorf("failed to unmarshal kustomization file: %w", err)
	}

	return nil
}

func (k *FluxKustomization) WriteToFile(file io.Writer) error {
	kustomizationRaw, err := yaml.Marshal(k)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization file: %w", err)
	}

	_, err = file.Write(kustomizationRaw)
	if err != nil {
		return fmt.Errorf("failed to write kustomization file: %w", err)
	}

	return nil
}

func (k *FluxKustomization) ApplyToCluster(ctx context.Context, cluster *clusters.Cluster) error {
	kMarshaled, err := yaml.Marshal(k)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization file: %w", err)
	}

	return util.ApplyManifests(ctx, cluster, kMarshaled)
}
