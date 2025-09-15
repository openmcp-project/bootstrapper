package cmd

import (
	"github.com/openmcp-project/bootstrapper/internal/log"
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"

	"github.com/spf13/cobra"
)

// ocmTransferCmd represents the "ocm transfer componentversion" command
var ocmTransferCmd = &cobra.Command{
	Use:   "ocm-transfer source target",
	Short: "Transfer an OCM component from a source to a target location",
	Long:  `Transfers the specified OCM component version from the source location to the target location.`,
	Aliases: []string{
		"transfer",
	},
	Args: cobra.ExactArgs(2),
	ArgAliases: []string{
		"source",
		"target",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		log := log.GetLogger()

		log.Debugf("Executing ocm-transfer with source: %s, target: %s", args[0], args[1])

		transferCommands := []string{
			"transfer",
			"componentversion",
		}

		transferArgs := []string{
			"--recursive",
			"--copy-resources",
			"--copy-sources",
			args[0], // source
			args[1], // target
		}

		return ocmcli.Execute(cmd.Context(), transferCommands, transferArgs, cmd.Flag(FlagOcmConfig).Value.String())
	},
}

func init() {
	RootCmd.AddCommand(ocmTransferCmd)

	ocmTransferCmd.PersistentFlags().String(FlagOcmConfig, "", "ocm configuration file")
}
