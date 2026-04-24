package main

import (
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/openmcp-project/bootstrapper/cmd"
)

func main() {
	if len(os.Args) < 2 {
		panic("documentation folder path required as argument")
	}
	rootCommand := cmd.RootCmd
	rootCommand.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(rootCommand, os.Args[1]); err != nil {
		panic(err)
	}
}
