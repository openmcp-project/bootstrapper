package eso_deployer

import (
	"context"
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openmcp-project/bootstrapper/internal/component"
	cfg "github.com/openmcp-project/bootstrapper/internal/config"
	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
	"github.com/openmcp-project/bootstrapper/internal/scheme"
)

func TestEsoDeployer_DeployWithComponentManager(t *testing.T) {
	platformClient := fake.NewClientBuilder().
		WithScheme(scheme.NewFluxScheme()).
		Build()
	platformCluster := clusters.NewTestClusterFromClient("platform", platformClient)
	namespace := flux_deployer.FluxSystemNamespace

	config := &cfg.BootstrapperConfig{
		Component:            cfg.Component{},
		DeploymentRepository: cfg.DeploymentRepository{},
		Providers:            cfg.Providers{},
		ImagePullSecrets:     nil,
		Environment:          "test",
	}

	d := NewEsoDeployer(config, "", platformCluster, logging.GetLogger())

	// Initial
	initial := &component.MockComponentManager{
		ComponentPath: "./testdata/component_1.yaml",
		TemplatesPath: "",
	}

	err := d.DeployWithComponentManager(context.Background(), initial)
	assert.NoError(t, err, "Error deploying eso controllers")

	chartRepo := &sourcev1.OCIRepository{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: esoChartRepoName, Namespace: namespace}, chartRepo)
	assert.NoError(t, err, "Error getting chart repo")
	assert.Equal(t, namespace, chartRepo.Namespace, "Repo namespace does not match expected namespace")
	assert.Equal(t, "external-secrets-v1.0.0", chartRepo.Spec.Reference.Tag, "Tag does not match expected tag")

	imgRepo := &sourcev1.OCIRepository{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: esoImageRepoName, Namespace: namespace}, imgRepo)
	assert.NoError(t, err, "Error getting image repo")
	assert.Equal(t, namespace, imgRepo.Namespace, "Repo namespace does not match expected namespace")
	assert.Equal(t, "external-secrets-v1.0.0", imgRepo.Spec.Reference.Tag, "Tag does not match expected tag")

	helmRelease := &helmv2.HelmRelease{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: esoHelmReleaseName, Namespace: namespace}, helmRelease)
	assert.NoError(t, err, "Error getting eso helm release")
	assert.Equal(t, namespace, helmRelease.Namespace, "HelmRelease namespace does not match expected namespace")
	assert.Equal(t, esoChartRepoName, helmRelease.Spec.ChartRef.Name, "ChartRef name does not match expected name")
	assert.Equal(t, "eso", helmRelease.Spec.ReleaseName, "ReleaseName does not match expected name")

	// Updated
	updated := &component.MockComponentManager{
		ComponentPath: "./testdata/component_2.yaml",
		TemplatesPath: "",
	}

	err = d.DeployWithComponentManager(context.Background(), updated)
	assert.NoError(t, err, "Error deploying eso controllers")

	chartRepo = &sourcev1.OCIRepository{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: esoChartRepoName, Namespace: namespace}, chartRepo)
	assert.NoError(t, err, "Error getting chart repo")
	assert.Equal(t, namespace, chartRepo.Namespace, "Repo namespace does not match expected namespace")
	assert.Equal(t, "external-secrets-v2.0.0", chartRepo.Spec.Reference.Tag, "Tag does not match expected tag")

	imgRepo = &sourcev1.OCIRepository{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: esoImageRepoName, Namespace: namespace}, imgRepo)
	assert.NoError(t, err, "Error getting image repo")
	assert.Equal(t, namespace, imgRepo.Namespace, "Repo namespace does not match expected namespace")
	assert.Equal(t, "external-secrets-v2.0.0", imgRepo.Spec.Reference.Tag, "Tag does not match expected tag")

	helmRelease = &helmv2.HelmRelease{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: esoHelmReleaseName, Namespace: namespace}, helmRelease)
	assert.NoError(t, err, "Error getting eso helm release")
	assert.Equal(t, namespace, helmRelease.Namespace, "HelmRelease namespace does not match expected namespace")
	assert.Equal(t, esoChartRepoName, helmRelease.Spec.ChartRef.Name, "ChartRef name does not match expected name")
	assert.Equal(t, "eso", helmRelease.Spec.ReleaseName, "ReleaseName does not match expected name")

}
