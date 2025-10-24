package component

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

// MockComponentManager is a mock implementation of the ComponentManager interface for testing purposes.
type MockComponentManager struct {
	ComponentPath string
	TemplatesPath string
}

var _ ComponentManager = (*MockComponentManager)(nil)

func (m MockComponentManager) GetComponentWithImageResources(_ context.Context, _ string) (*ocmcli.ComponentVersion, error) {
	return loadComponentVersion(m.ComponentPath)
}

func (m MockComponentManager) DownloadTemplatesResource(_ context.Context, downloadDir string) error {
	return util.CopyDir(m.TemplatesPath, downloadDir)
}

func loadComponentVersion(path string) (*ocmcli.ComponentVersion, error) {
	cv := &ocmcli.ComponentVersion{}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading component version from file %s: %w", path, err)
	}
	err = yaml.Unmarshal(content, cv)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling component version from file %s: %w", path, err)
	}
	return cv, nil
}
