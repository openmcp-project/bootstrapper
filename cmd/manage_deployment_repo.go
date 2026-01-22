package cmd

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/openmcp-project/bootstrapper/internal/config"

	"github.com/openmcp-project/bootstrapper/internal/util"

	deploymentrepo "github.com/openmcp-project/bootstrapper/internal/deployment-repo"
	"github.com/openmcp-project/bootstrapper/internal/log"
)

const (
	FlagExtraManifestDir          = "extra-manifest-dir"
	FlagKustomizationPatches      = "kustomization-patches"
	FlagDisableGitPush            = "disable-git-push"
	FlagDisableKustomizationApply = "disable-kustomization-apply"
	FlagDryRun                    = "dry-run"
	FlagPrintKustomized           = "print-kustomized"
	FlagCommitMessage             = "commit-message"
)

type LogWriter struct{}

func (w LogWriter) Write(p []byte) (n int, err error) {
	logger := log.GetLogger()
	logger.Debugf("Git progress: %s", string(p))
	return len(p), nil
}

// manageDeploymentRepoCmd represents the manageDeploymentRepo command
var manageDeploymentRepoCmd = &cobra.Command{
	Use:   "manage-deployment-repo",
	Short: "Updates the openMCP deployment specification in the specified Git repository",
	Long: `Updates the openMCP deployment specification in the specified Git repository.
The update is based on the specified component version.
openmcp-bootstrapper manage-deployment-repo <configFile>`,
	Args: cobra.ExactArgs(1),
	ArgAliases: []string{
		"configFile",
	},
	Example: `  openmcp-bootstrapper manage-deployment-repo "./config.yaml"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFilePath := args[0]
		logger := log.GetLogger()

		// disable controller-runtime logging
		controllerruntime.SetLogger(logr.Discard())

		disableGitPush, err := cmd.Flags().GetBool(FlagDisableGitPush)
		if err != nil {
			return fmt.Errorf("failed to parse disable-git-push flag: %w", err)
		}

		disableKustomizationApply, err := cmd.Flags().GetBool(FlagDisableKustomizationApply)
		if err != nil {
			return fmt.Errorf("failed to parse disable-kustomization-apply flag: %w", err)
		}

		dryRun, err := cmd.Flags().GetBool(FlagDryRun)
		if err != nil {
			return fmt.Errorf("failed to parse dry-run flag: %w", err)
		}

		printKustomized, err := cmd.Flags().GetBool(FlagPrintKustomized)
		if err != nil {
			return fmt.Errorf("failed to parse print-kustomized flag: %w", err)
		}

		if dryRun {
			logger.Info("Running in dry-run mode: no changes will be applied to the git repository or the target cluster")
			disableGitPush = true
			disableKustomizationApply = true
		}

		var targetCluster *clusters.Cluster
		if !disableKustomizationApply {
			targetCluster, err = util.GetCluster(cmd.Flag(FlagKubeConfig).Value.String(), "target-cluster", runtime.NewScheme())
			if err != nil {
				return fmt.Errorf("failed to get platform cluster: %w", err)
			}
		}

		config := &config.BootstrapperConfig{}
		err = config.ReadFromFile(configFilePath)
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
			targetCluster,
			cmd.Flag(FlagGitConfig).Value.String(),
			cmd.Flag(FlagOcmConfig).Value.String(),
			cmd.Flag(FlagExtraManifestDir).Value.String(),
			cmd.Flag(FlagKustomizationPatches).Value.String(),
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

		err = deploymentRepoManager.ApplyExtraManifests(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to apply extra manifests: %w", err)
		}

		err = deploymentRepoManager.UpdateResourcesKustomization()
		if err != nil {
			return fmt.Errorf("failed to update resources kustomization: %w", err)
		}

		if !disableGitPush {
			err = deploymentRepoManager.CommitAndPushChanges(cmd.Context(), cmd.Flag(FlagCommitMessage).Value.String())
			if err != nil {
				return fmt.Errorf("failed to commit and push changes: %w", err)
			}
		} else {
			logger.Info("Skipping pushing changes to git repository as per flag")
		}

		manifests, err := deploymentRepoManager.RunKustomize()
		if err != nil {
			return fmt.Errorf("failed to run kustomize: %w", err)
		}

		if !disableKustomizationApply {
			err = deploymentRepoManager.RunKustomizeAndApply(cmd.Context(), manifests)
			if err != nil {
				return fmt.Errorf("failed to run kustomize and apply: %w", err)
			}
		} else {
			logger.Info("Skipping applying kustomization to target cluster as per flag")
		}

		if printKustomized {
			logger.Info("Kustomized manifests:")
			err = util.PrintUnstructuredObjects(manifests, os.Stdout)
			if err != nil {
				return fmt.Errorf("failed to print kustomized manifests: %w", err)
			}
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(manageDeploymentRepoCmd)
	manageDeploymentRepoCmd.Flags().SortFlags = false
	manageDeploymentRepoCmd.Flags().String(FlagOcmConfig, "", "ocm configuration file")
	manageDeploymentRepoCmd.Flags().String(FlagGitConfig, "", "Git configuration file")
	manageDeploymentRepoCmd.Flags().String(FlagKubeConfig, "", "Kubernetes configuration file")
	manageDeploymentRepoCmd.Flags().String(FlagExtraManifestDir, "", "Directory containing extra manifests to apply")
	manageDeploymentRepoCmd.Flags().String(FlagKustomizationPatches, "", "YAML file containing kustomization patches to apply")
	manageDeploymentRepoCmd.Flags().Bool(FlagDisableGitPush, false, "If true, disables pushing changes to the git repository")
	manageDeploymentRepoCmd.Flags().Bool(FlagDisableKustomizationApply, false, "If true, disables applying the kustomization to the target cluster")
	manageDeploymentRepoCmd.Flags().Bool(FlagDryRun, false, "If true, performs a dry run without applying any changes to the git repo and the target cluster")
	manageDeploymentRepoCmd.Flags().Bool(FlagPrintKustomized, false, "If true, prints the kustomized manifests to stdout")
	manageDeploymentRepoCmd.Flags().String(FlagCommitMessage, "apply templates", "Commit message to use when pushing changes to the git repository")

	if err := manageDeploymentRepoCmd.MarkFlagRequired(FlagGitConfig); err != nil {
		panic(err)
	}
}
