package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cfg "github.com/openmcp-project/bootstrapper/internal/config"

	esodeployer "github.com/openmcp-project/bootstrapper/internal/eso-deployer"
	logging "github.com/openmcp-project/bootstrapper/internal/log"
	"github.com/openmcp-project/bootstrapper/internal/scheme"
	"github.com/openmcp-project/bootstrapper/internal/util"
)

// deployEsoCmd represents the deploy-eso command
var deployEsoCmd = &cobra.Command{
	Use:   "deploy-eso",
	Short: "Deploys External Secrets Operator controllers on the target cluster",
	Long:  "Deploys External Secrets Operator controllers on the target cluster",
	Args:  cobra.ExactArgs(1),
	ArgAliases: []string{
		"configFile",
	},
	Example: `  openmcp-bootstrapper deploy-eso "./config.yaml"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFilePath := args[0]
		config := &cfg.BootstrapperConfig{}
		err := config.ReadFromFile(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		log := logging.GetLogger()
		log.Info("Starting deployment of external secrets operator controllers.")

		targetCluster, err := util.GetCluster(cmd.Flag(FlagKubeConfig).Value.String(), "target-cluster", scheme.NewFluxScheme())
		if err != nil {
			return fmt.Errorf("failed to get platform cluster: %w", err)
		}

		if err = esodeployer.NewEsoDeployer(config, cmd.Flag(FlagOcmConfig).Value.String(), targetCluster, log).Deploy(cmd.Context()); err != nil {
			return fmt.Errorf("failed deploying eso: %w", err)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(deployEsoCmd)
	deployEsoCmd.Flags().SortFlags = false
	deployEsoCmd.Flags().String(FlagOcmConfig, "", "OCM configuration file")
	deployEsoCmd.Flags().String(FlagKubeConfig, "", "Kubernetes configuration file")
}
