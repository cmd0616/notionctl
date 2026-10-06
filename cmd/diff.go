package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/radityajay/notionctl/internal/config"
	"github.com/radityajay/notionctl/internal/engine"
	"github.com/radityajay/notionctl/internal/notion"
	"github.com/radityajay/notionctl/internal/state"

	// Register all property types.
	_ "github.com/radityajay/notionctl/internal/property"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare remote Notion databases against local config",
	Long: `Fetches each database from Notion and compares it against your YAML config.
Shows properties that have drifted (added/removed/changed in Notion but not in config).

Requires NOTION_TOKEN and databases to be previously applied (have state IDs).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := os.Getenv("NOTION_TOKEN")
		if token == "" {
			return fmt.Errorf("NOTION_TOKEN environment variable is required\nGet one at https://www.notion.so/my-integrations")
		}

		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		st, err := state.Load(".")
		if err != nil {
			return err
		}

		client := notion.NewClient(token)
		eng := engine.New(cfg, st, client, ".")

		results, err := eng.Diff()
		if err != nil {
			return err
		}

		fmt.Print(engine.FormatDiff(results))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
