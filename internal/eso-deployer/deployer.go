package eso_deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"

	"github.com/openmcp-project/bootstrapper/internal/component"
	cfg "github.com/openmcp-project/bootstrapper/internal/config"

	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	"github.com/openmcp-project/bootstrapper/internal/util"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type EsoDeployer struct {
	Config *cfg.BootstrapperConfig

	// OcmConfigPath is the path to the OCM configuration file
	OcmConfigPath string

	platformCluster *clusters.Cluster
	log             *logrus.Logger
}

func NewEsoDeployer(config *cfg.BootstrapperConfig, ocmConfigPath string, platformCluster *clusters.Cluster, log *logrus.Logger) *EsoDeployer {
	return &EsoDeployer{
		Config:          config,
		OcmConfigPath:   ocmConfigPath,
		platformCluster: platformCluster,
		log:             log,
	}
}

func (d *EsoDeployer) Deploy(ctx context.Context) error {
	componentManager, err := component.NewComponentManager(ctx, d.Config, d.OcmConfigPath)
	if err != nil {
		return fmt.Errorf("error creating component manager: %w", err)
	}

	return d.DeployWithComponentManager(ctx, componentManager)
}

func (d *EsoDeployer) DeployWithComponentManager(ctx context.Context, componentManager component.ComponentManager) error {
	d.log.Info("Getting OCM component containing ESO resources.")
	esoComponent, err := componentManager.GetComponentWithImageResources(ctx, "external-secrets-operator-image")
	if err != nil {
		return fmt.Errorf("failed to get external-secrets-operator-image component: %w", err)
	}

	esoChartRes, err := esoComponent.GetResource("external-secrets-operator-chart")
	if err != nil {
		return fmt.Errorf("failed to get external-secrets-operator-chart resource: %w", err)
	}
	d.log.Info("Deploying OCIRepo for ESO chart.")
	if err = deployRepo(ctx, d, esoChartRes, esoChartRepoName); err != nil {
		return fmt.Errorf("failed to create helm chart repo: %w", err)
	}

	esoImageRes, err := esoComponent.GetResource("external-secrets-operator-image")
	if err != nil {
		return fmt.Errorf("failed to get external-secrets-operator-image resource: %w", err)
	}
	d.log.Info("Deploying OCIRepo for ESO image.")
	if err = deployRepo(ctx, d, esoImageRes, esoImageRepoName); err != nil {
		return fmt.Errorf("failed to create helm image repo: %w", err)
	}

	d.log.Info("Deploying HelmRelease for ESO.")
	if err = deployHelmRelease(ctx, d, esoImageRes); err != nil {
		return fmt.Errorf("failed to deploy helm release: %w", err)
	}

	d.log.Info("Done.")
	return nil
}

func deployHelmRelease(ctx context.Context, d *EsoDeployer, res *ocmcli.Resource) error {
	name, _, _, err := util.ParseImageVersionAndTag(*res.Access.ImageReference)
	if err != nil {
		return fmt.Errorf("failed to parse image resource: %w", err)
	}

	values := map[string]any{
		"image": map[string]any{"repository": name},
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal ESO Helm values: %w", err)
	}
	jsonVals := &apiextensionsv1.JSON{Raw: encoded}

	helmRelease := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      esoHelmReleaseName,
			Namespace: flux_deployer.FluxSystemNamespace,
		},
		Spec: helmv2.HelmReleaseSpec{
			ChartRef: &helmv2.CrossNamespaceSourceReference{
				Kind:      "OCIRepository",
				Name:      esoChartRepoName,
				Namespace: flux_deployer.FluxSystemNamespace,
			},
			ReleaseName:     "eso",
			TargetNamespace: esoNamespace,
			Install: &helmv2.Install{
				CreateNamespace: true,
			},
			Values: jsonVals,
		},
	}
	return util.CreateOrUpdate(ctx, d.platformCluster, helmRelease)
}

func deployRepo(ctx context.Context, d *EsoDeployer, res *ocmcli.Resource, repoName string) error {
	imageName, tag, digest, err := util.ParseImageVersionAndTag(*res.Access.ImageReference)
	if err != nil {
		return err
	}

	ociRepo := &sourcev1.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      repoName,
			Namespace: flux_deployer.FluxSystemNamespace,
		},
		Spec: sourcev1.OCIRepositorySpec{
			URL: fmt.Sprintf("oci://%s", imageName),
			Reference: &sourcev1.OCIRepositoryRef{
				Tag:    tag,
				Digest: digest,
			},
			Timeout: &metav1.Duration{Duration: 1 * time.Minute},
		},
	}
	return util.CreateOrUpdate(ctx, d.platformCluster, ociRepo)
}
