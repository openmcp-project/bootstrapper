package flux_deployer_test

import (
	"testing"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
)

func TestDeployFluxController(t *testing.T) {
	rootComponentVersion1 := LoadComponentVersion(t, "./testdata/01/root-component-version-1.yaml")
	rootComponentVersion2 := LoadComponentVersion(t, "./testdata/01/root-component-version-2.yaml")
	downloadDir := "./testdata/01/download_dir"

	platformClient := fake.NewClientBuilder().Build()
	platformCluster := clusters.NewTestClusterFromClient("platform", platformClient)
	namespace := "flux-system-test"

	d := flux_deployer.NewFluxDeployer("", "", "",
		"", "", ocmcli.NoOcmConfig, "", namespace, "", platformCluster, logging.GetLogger())

	// Create a deployment
	err := d.DeployFluxControllers(t.Context(), rootComponentVersion1, downloadDir)
	assert.NoError(t, err, "Error deploying flux controllers")

	deployment := &v1.Deployment{}
	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "source-controller", Namespace: namespace}, deployment)
	assert.NoError(t, err, "Error getting source-controller deployment")
	assert.Equal(t, namespace, deployment.Namespace, "Deployment namespace does not match expected namespace")
	assert.Equal(t, "test-source-controller-image:v0.0.1", deployment.Spec.Template.Spec.Containers[0].Image, "Deployment image does not match expected image")

	// Update the deployment
	err = d.DeployFluxControllers(t.Context(), rootComponentVersion2, downloadDir)
	assert.NoError(t, err, "Error updating flux controllers")

	err = platformClient.Get(t.Context(), client.ObjectKey{Name: "source-controller", Namespace: namespace}, deployment)
	assert.NoError(t, err, "Error getting source-controller deployment")
	assert.Equal(t, namespace, deployment.Namespace, "Deployment namespace does not match expected namespace")
	assert.Equal(t, "test-source-controller-image:v0.0.2", deployment.Spec.Template.Spec.Containers[0].Image, "Deployment image does not match expected image")
}
