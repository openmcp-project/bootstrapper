package flux_deployer_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
)

func TestKustomize(t *testing.T) {
	resourcesYaml, err := flux_deployer.Kustomize("./testdata/02/dev/fluxcd")
	assert.NoError(t, err, "Error running kustomization")
	assert.NotNil(t, resourcesYaml, "Resources yaml should not be nil")

	resources, err := flux_deployer.ParseManifests(bytes.NewReader(resourcesYaml))
	assert.NoError(t, err, "Error parsing manifests")
	assert.Len(t, resources, 3, "There should be 3 resources")

	deploy := getResource(resources, "Deployment", "source-controller", "flux-system")
	assert.NotNil(t, deploy, "Deployment should be present")

	repo := getResource(resources, "GitRepository", "environments", "flux-system")
	assert.NotNil(t, repo, "GitRepository should be present")
	branch, found, err := unstructured.NestedString(repo.Object, "spec", "ref", "branch")
	assert.NoError(t, err, "Error getting GitRepository branch")
	assert.True(t, found, "GitRepository branch should be found")
	assert.Equal(t, "dev", branch, "GitRepository branch should have the expected value")

	kustomization := getResource(resources, "Kustomization", "flux-system", "flux-system")
	assert.NotNil(t, kustomization, "Kustomization should be present")
	path, found, err := unstructured.NestedString(kustomization.Object, "spec", "path")
	assert.NoError(t, err, "Error getting Kustomization path")
	assert.True(t, found, "Kustomization path should be found")
	assert.Equal(t, "dev/fluxcd", path, "Kustomization path should have the expected value")
}

func getResource(resources []*unstructured.Unstructured, kind, name, namespace string) *unstructured.Unstructured {
	for _, res := range resources {
		if res.GetKind() == kind && res.GetName() == name && res.GetNamespace() == namespace {
			return res
		}
	}
	return nil
}
