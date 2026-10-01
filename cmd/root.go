package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile string
)

var rootCmd = &cobra.Command{
	Use:   "notionctl",
	Short: "Declarative Notion database management",
	Long: `notionctl lets you define Notion databases in YAML and sync them
to your workspace. Manage relations by name, not by ID.

  notionctl init          Generate a starter YAML config
  notionctl plan          Preview changes before applying
  notionctl apply         Create/update databases in Notion`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "notionctl.yaml", "path to config file")
}
