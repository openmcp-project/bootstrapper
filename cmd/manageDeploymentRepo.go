package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	deploymentrepo "github.com/openmcp-project/bootstrapper/internal/deployment-repo"
	"github.com/openmcp-project/bootstrapper/internal/log"
)

const (
	FlagGitConfig = "git-config"
	FlagOcmConfig = "ocm-config"
)

type LogWriter struct{}

func (w LogWriter) Write(p []byte) (n int, err error) {
	logger := log.GetLogger()
	logger.Debugf("Git progress: %s", string(p))
	return len(p), nil
}

// manageDeploymentRepoCmd represents the manageDeploymentRepo command
var manageDeploymentRepoCmd = &cobra.Command{
	Use:   "manageDeploymentRepo",
	Short: "Updates the openMCP deployment specification in the specified Git repository",
	Long: `Updates the openMCP deployment specification in the specified Git repository.
The update is based on the specified component version.
openmcp-bootstrapper manageDeploymentRepo <configFile>`,
	Args: cobra.ExactArgs(1),
	ArgAliases: []string{
		"configFile",
	},
	Example: `  openmcp-bootstrapper manageDeploymentRepo "./config.yaml"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFilePath := args[0]

		config := &deploymentrepo.DeploymentRepoConfig{}
		err := config.ReadFromFile(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		config.SetDefaults()
		err = config.Validate()
		if err != nil {
			return fmt.Errorf("invalid config file: %w", err)
		}

		deploymentRepoManager, err := deploymentrepo.NewDeploymentRepoManager(
			config,
			cmd.Flag(FlagGitConfig).Value.String(),
			cmd.Flag(FlagOcmConfig).Value.String(),
		).Initialize(cmd.Context())

		defer func() {
			deploymentRepoManager.Cleanup()
		}()

		if err != nil {
			return fmt.Errorf("failed to initialize deployment repo manager: %w", err)
		}

		err = deploymentRepoManager.ApplyTemplates(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to apply templates: %w", err)
		}

		err = deploymentRepoManager.ApplyProviders(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to apply providers: %w", err)
		}

		err = deploymentRepoManager.ApplyCustomResourceDefinitions(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to apply custom resource definitions: %w", err)
		}

		err = deploymentRepoManager.UpdateResourcesKustomization()
		if err != nil {
			return fmt.Errorf("failed to update resources kustomization: %w", err)
		}

		err = deploymentRepoManager.CommitAndPushChanges(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to commit and push changes: %w", err)
		}

		err = deploymentRepoManager.RunKustomizeAndApply(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to run kustomize and apply: %w", err)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(manageDeploymentRepoCmd)
	manageDeploymentRepoCmd.Flags().SortFlags = false
	manageDeploymentRepoCmd.Flags().String(FlagOcmConfig, "", "ocm configuration file")
	manageDeploymentRepoCmd.Flags().String(FlagGitConfig, "", "Git configuration file")

	if err := manageDeploymentRepoCmd.MarkFlagRequired(FlagGitConfig); err != nil {
		panic(err)
	}
}
