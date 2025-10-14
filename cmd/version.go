package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/yaml"

	ocmcli "github.com/openmcp-project/bootstrapper/internal/ocm-cli"

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
		ownVersion := version.GetVersion()

		if ownVersion == nil {
			fmt.Println("Version information not available")
			return
		}

		printVersion(ownVersion, "openMCP Bootstrapper CLI")

		var ocmVersion apimachineryversion.Info
		out, err := ocmcli.ExecuteOutput(cmd.Context(), []string{"version"}, nil, ocmcli.NoOcmConfig)
		if err != nil {
			fmt.Printf("Error retrieving ocm-cli version: %ownVersion\n", err)
			return
		}

		err = yaml.Unmarshal(out, &ocmVersion)
		if err != nil {
			fmt.Printf("Error parsing ocm-cli version output: %ownVersion\n", err)
			return
		}

		printVersion(&ocmVersion, "OCM CLI")
	},
}

func printVersion(v *apimachineryversion.Info, header string) {
	fmt.Printf("\n%s\n", header)
	fmt.Printf("========================\n")
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
	fmt.Printf("===================\n")
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
