package flux_deployer_test

import (
	"testing"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cfg "github.com/openmcp-project/bootstrapper/internal/config"
	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
)

func TestDeployFluxController(t *testing.T) {

	platformClient := fake.NewClientBuilder().Build()
	platformCluster := clusters.NewTestClusterFromClient("platform", platformClient)
	namespace := flux_deployer.FluxSystemNamespace

	config := &cfg.BootstrapperConfig{
		Component: cfg.Component{
			OpenMCPComponentLocation: "./testdata/01/root-component-version-1.yaml",
		},
		DeploymentRepository: cfg.DeploymentRepository{},
		Providers:            cfg.Providers{},
		ImagePullSecrets:     nil,
		Environment:          "test",
	}

	d := flux_deployer.NewFluxDeployer(config, "", ocmcli.NoOcmConfig, platformCluster, logging.GetLogger())

	// Initial deployment
	componentManager1 := &MockComponentManager{
		ComponentPath: "./testdata/01/component_1.yaml",
		TemplatesPath: "./testdata/01/fluxcd_resource",
	}

	err := d.DeployWithComponentManager(t.Context(), componentManager1)
	assert.NoError(t, err, "Error deploying flux controllers")
	deployment := &v1.Deployment{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "source-controller", Namespace: namespace}, deployment)
	assert.NoError(t, err, "Error getting source-controller deployment")
	assert.Equal(t, namespace, deployment.Namespace, "Deployment namespace does not match expected namespace")
	assert.Equal(t, "ghcr.io/fluxcd/source-controller:v1.0.0", deployment.Spec.Template.Spec.Containers[0].Image, "Deployment image does not match expected image")

	// Update deployment
	componentManager2 := &MockComponentManager{
		ComponentPath: "./testdata/01/component_2.yaml",
		TemplatesPath: "./testdata/01/fluxcd_resource",
	}

	err = d.DeployWithComponentManager(t.Context(), componentManager2)
	assert.NoError(t, err, "Error deploying flux controllers")
	deployment = &v1.Deployment{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "source-controller", Namespace: namespace}, deployment)
	assert.NoError(t, err, "Error getting source-controller deployment")
	assert.Equal(t, namespace, deployment.Namespace, "Deployment namespace does not match expected namespace")
	assert.Equal(t, "ghcr.io/fluxcd/source-controller:v2.0.0", deployment.Spec.Template.Spec.Containers[0].Image, "Deployment image does not match expected image")
}
