package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	cfg "github.com/openmcp-project/bootstrapper/internal/config"
	"github.com/openmcp-project/bootstrapper/internal/flux_deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

// deployFluxCmd represents the "deploy flux" command
var deployFluxCmd = &cobra.Command{
	Use:   "deploy-flux",
	Short: "Deploys Flux controllers on the platform cluster, and establishes synchronization with a Git repository",
	Long:  `Deploys Flux controllers on the platform cluster, and establishes synchronization with a Git repository.`,
	Args:  cobra.ExactArgs(1),
	ArgAliases: []string{
		"configFile",
	},
	Example: `  openmcp-bootstrapper deploy-flux "./config.yaml"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFilePath := args[0]

		log := logging.GetLogger()
		log.Infof("Starting deployment of Flux controllers with config file: %s.", configFilePath)

		// Configuration
		config := &cfg.BootstrapperConfig{}
		err := config.ReadFromFile(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		config.SetDefaults()
		err = config.Validate()
		if err != nil {
			return fmt.Errorf("invalid config file: %w", err)
		}

		// Platform cluster
		scheme := runtime.NewScheme()
		if err := v1.AddToScheme(scheme); err != nil {
			return fmt.Errorf("error adding corev1 to scheme: %w", err)
		}

		platformCluster, err := util.GetCluster(cmd.Flag(FlagKubeConfig).Value.String(), "platform", scheme)
		if err != nil {
			return fmt.Errorf("failed to get platform cluster: %w", err)
		}
		if err := platformCluster.InitializeRESTConfig(); err != nil {
			return fmt.Errorf("error initializing REST config for platform cluster: %w", err)
		}
		if err := platformCluster.InitializeClient(nil); err != nil {
			return fmt.Errorf("error initializing client for platform cluster: %w", err)
		}

		d := flux_deployer.NewFluxDeployer(config, cmd.Flag(FlagGitConfig).Value.String(), cmd.Flag(FlagOcmConfig).Value.String(), platformCluster, log)
		if err = d.Deploy(cmd.Context()); err != nil {
			log.Errorf("Deployment of flux controllers failed: %v", err)
			return err
		}

		log.Info("Deployment of flux controllers completed")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(deployFluxCmd)
	deployFluxCmd.Flags().SortFlags = false
	deployFluxCmd.Flags().String(FlagOcmConfig, "", "OCM configuration file")
	deployFluxCmd.Flags().String(FlagGitConfig, "", "Git credentials configuration file that configures basic auth or ssh private key. This will be used in the fluxcd GitSource for spec.secretRef to authenticate against the deploymentRepository. If not set, no authentication will be configured.")
	deployFluxCmd.Flags().String(FlagKubeConfig, "", "Kubernetes configuration file")

	if err := deployFluxCmd.MarkFlagRequired(FlagGitConfig); err != nil {
		panic(err)
	}
}
