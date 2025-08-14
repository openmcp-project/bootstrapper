package cmd

import (
	"fmt"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/spf13/cobra"

	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
)

const (
	flagOCMConfig       = "ocm-config"
	flagGitCredentials  = "git-credentials"
	flagKubeconfig      = "kubeconfig"
	flagFluxCDNamespace = "fluxcd-namespace"
)

// deployFluxCmd represents the "deploy flux" command
var deployFluxCmd = &cobra.Command{
	Use:   "deploy-flux source target",
	Short: "Transfer an OCM component from a source to a target location",
	Long:  `Transfers the specified OCM component version from the source location to the target location.`,
	Args:  cobra.ExactArgs(5),
	ArgAliases: []string{
		"component-location",
		"deployment-templates",
		"deployment-repository",
		"deployment-repository-branch",
		"deployment-repository-path",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logging.GetLogger()
		log.Infof("Starting flux deployment with component-location: %s, deployment-templates: %s, "+
			"deployment-repository: %s, deployment-repository-branch: %s, deployment-repository-path: %s",
			args[0], args[1], args[2], args[3], args[4])

		platformKubeconfig := cmd.Flag(flagKubeconfig).Value.String()
		platformCluster := clusters.New("platform").WithConfigPath(platformKubeconfig)
		if err := platformCluster.InitializeRESTConfig(); err != nil {
			return fmt.Errorf("error initializing REST config for platform cluster: %w", err)
		}
		if err := platformCluster.InitializeClient(nil); err != nil {
			return fmt.Errorf("error initializing client for platform cluster: %w", err)
		}

		d := flux_deployer.NewFluxDeployer(args[0], args[1], args[2], args[3], args[4],
			cmd.Flag(flagOCMConfig).Value.String(),
			cmd.Flag(flagGitCredentials).Value.String(),
			cmd.Flag(flagFluxCDNamespace).Value.String(),
			platformKubeconfig,
			platformCluster, log)
		err := d.Deploy(cmd.Context())
		if err != nil {
			log.Errorf("Flux deployment failed: %v", err)
			return err
		}

		log.Info("Flux deployment completed")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(deployFluxCmd)

	deployFluxCmd.Flags().StringP(flagOCMConfig, "c", "", "ocm configuration file")
	deployFluxCmd.Flags().StringP(flagGitCredentials, "g", "", "git credentials configuration file that configures basic auth, personal access token, ssh private key. This will be used in the fluxcd GitSource for spec.secretRef to authenticate against the deploymentRepository. If not set, no authentication will be configured.")
	deployFluxCmd.Flags().StringP(flagKubeconfig, "k", "", "kubeconfig of the Kubernetes cluster on which the flux deployment will be created/updated. If not set, the current context will be used.")
	deployFluxCmd.Flags().StringP(flagFluxCDNamespace, "n", "", "namespace on the Kubernetes cluster in which the namespaced fluxcd resources will be deployed. Default 'flux-system'.")
}
