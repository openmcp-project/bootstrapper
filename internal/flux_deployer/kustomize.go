package flux_deployer

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// Kustomize runs kustomize on the given directory and returns the resulting yaml as a byte slice.
func Kustomize(dir string) ([]byte, error) {
	fmt.Println("kustomize")

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
