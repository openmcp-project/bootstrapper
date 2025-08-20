package cmd

import (
	"os"

	"github.com/openmcp-project/bootstrapper/internal/log"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "openmcp-bootstrapper",
	Short: "The openMCP bootstrapper CLI",
	Long: `The openMCP bootstrapper CLI is a command-line interface
for bootstrapping and updating openMCP landscapes.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringP("verbosity", "v", "info", "Set the verbosity level (panic, fatal, error, warn, info, debug, trace)")
	cobra.OnInitialize(func() {
		verbosity, err := RootCmd.PersistentFlags().GetString("verbosity")
		if err != nil {
			log.InitLogger("info")
			return
		}
		log.InitLogger(verbosity)
	})
}
