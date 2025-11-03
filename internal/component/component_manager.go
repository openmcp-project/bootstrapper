package component

import (
	"context"

	cfg "github.com/openmcp-project/bootstrapper/internal/config"
	ocm_cli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"
)

// ComponentManager bundles the OCM logic required by the FluxDeployer.
type ComponentManager interface {
	GetComponentsWithImageResources(ctx context.Context, resourceName string) ([]ocm_cli.ComponentVersion, error)
	DownloadTemplatesResource(ctx context.Context, downloadDir string) error
}

type ComponentManagerImpl struct {
	Config          *cfg.BootstrapperConfig
	OCMConfigPath   string
	ComponentGetter *ocm_cli.ComponentGetter
}

var _ ComponentManager = (*ComponentManagerImpl)(nil)

func NewComponentManager(ctx context.Context, config *cfg.BootstrapperConfig, ocmConfigPath string) (ComponentManager, error) {
	m := &ComponentManagerImpl{
		Config:          config,
		OCMConfigPath:   ocmConfigPath,
		ComponentGetter: ocm_cli.NewComponentGetter(config.Component.OpenMCPComponentLocation, config.Component.FluxcdTemplateResourcePath, ocmConfigPath),
	}

	if err := m.ComponentGetter.InitializeComponents(ctx); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *ComponentManagerImpl) GetComponentsWithImageResources(ctx context.Context, resourceName string) ([]ocm_cli.ComponentVersion, error) {
	return m.ComponentGetter.GetComponentVersionsForResourceRecursive(ctx, m.ComponentGetter.RootComponentVersion(), resourceName)
}

func (m *ComponentManagerImpl) DownloadTemplatesResource(ctx context.Context, downloadDir string) error {
	return m.ComponentGetter.DownloadTemplatesResource(ctx, downloadDir)
}
