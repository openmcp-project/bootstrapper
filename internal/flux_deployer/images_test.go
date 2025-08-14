package flux_deployer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
)

func TestGetFluxCDImages(t *testing.T) {
	cv := LoadComponentVersion(t, "./testdata/01/root-component-version-1.yaml")

	imageMap, err := flux_deployer.GetFluxCDImages(cv)
	assert.NoError(t, err, "error getting images")
	img, found := imageMap[flux_deployer.FluxcdSourceController]
	assert.True(t, found, "fluxcd source controller image should be found")
	assert.Equal(t, "test-source-controller-image:v0.0.1", img, "fluxcd source controller image should match")
	img, found = imageMap[flux_deployer.FluxcdHelmController]
	assert.True(t, found, "fluxcd helm controller image should be found")
	assert.Equal(t, "test-helm-controller-image:v0.0.1", img, "fluxcd helm controller image should match")
	img, found = imageMap[flux_deployer.FluxcdKustomizeController]
	assert.True(t, found, "fluxcd kustomize controller image should be found")
	assert.Equal(t, "test-kustomize-controller-image:v0.0.1", img, "fluxcd kustomize controller image should match")
}

func TestGetFluxCDImagesError(t *testing.T) {
	cv := LoadComponentVersion(t, "./testdata/01/root-component-version-1.yaml")
	cv.Component.Resources = cv.Component.Resources[0:1] // remove all but one resource

	_, err := flux_deployer.GetFluxCDImages(cv)
	assert.Error(t, err, "expected error getting images")
}
