package flux_deployer

import (
	"fmt"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
)

// GetFluxCDImages retrieves the images of the FluxCD controllers from the root component version.
func GetFluxCDImages(rootCV *ocmcli.ComponentVersion) (map[string]any, error) {
	return GetImages(rootCV, FluxCDHelmControllerResourceName, FluxCDKustomizationControllerResourceName, FluxCDSourceControllerResourceName)
}

// GetImages retrieves the images references for a list of resources of a component version.
// It returns a map with the resource names as keys and their image references as values.
// The returned map can be used to build the values for templating.
func GetImages(cv *ocmcli.ComponentVersion, resourceNames ...string) (map[string]any, error) {
	images := map[string]any{}

	for _, resourceName := range resourceNames {
		res, err := cv.GetResource(resourceName)
		if err != nil {
			return nil, err
		}
		if res == nil || res.Access.ImageReference == nil {
			return nil, fmt.Errorf("image reference of resource %s not found", resourceName)
		}
		images[resourceName] = *res.Access.ImageReference
	}

	return images, nil
}
