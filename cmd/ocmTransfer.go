package cmd

import (
	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"

	"github.com/spf13/cobra"
)

// ocmTransferCmd represents the "ocm transfer componentversion" command
var ocmTransferCmd = &cobra.Command{
	Use:   "ocmTransfer source destination",
	Short: "Transfer an OCM component from a source to a destination",
	Long:  `Transfers the specified OCM component version from the source location to the destination location.`,
	Aliases: []string{
		"transfer",
	},
	Args: cobra.ExactArgs(2),
	ArgAliases: []string{
		"source",
		"destination",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		transferCommands := []string{
			"transfer",
			"componentversion",
		}

		transferArgs := []string{
			"--recursive",
			"--copy-resources",
			"--copy-sources",
			args[0], // source
			args[1], // destination
		}

		return ocmcli.Execute(cmd.Context(), transferCommands, transferArgs, cmd.Flag("config").Value.String())
	},
}

func init() {
	rootCmd.AddCommand(ocmTransferCmd)

	ocmTransferCmd.PersistentFlags().StringP("config", "c", "", "ocm configuration file")
}
