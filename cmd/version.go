package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set via ldflags at build time:
//
//	go build -ldflags "-X github.com/radityajayantara/notionctl/cmd.version=1.0.0"
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of notionctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("notionctl %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
