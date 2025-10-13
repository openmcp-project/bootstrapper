package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openmcp-project/bootstrapper/internal/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long: `Print the version information of the openMCP bootstrapper CLI.

This command displays detailed version information including the build version,
Git commit, build date, Go version, and platform information.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func printVersion() {
	v := version.GetVersion()

	if v == nil {
		fmt.Println("Version information not available")
		return
	}

	fmt.Printf("Version:      %s\n", v.GitVersion)
	if v.GitCommit != "" {
		fmt.Printf("Git Commit:   %s\n", v.GitCommit)
	}
	if v.GitTreeState != "" {
		fmt.Printf("Git State:    %s\n", v.GitTreeState)
	}
	if v.BuildDate != "" {
		fmt.Printf("Build Date:   %s\n", v.BuildDate)
	}
	fmt.Printf("Go Version:   %s\n", v.GoVersion)
	fmt.Printf("Compiler:     %s\n", v.Compiler)
	fmt.Printf("Platform:     %s\n", v.Platform)
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
