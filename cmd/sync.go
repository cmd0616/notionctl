package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/radityajay/notionctl/internal/config"
	"github.com/radityajay/notionctl/internal/importer"
	"github.com/radityajay/notionctl/internal/notion"
	"github.com/radityajay/notionctl/internal/state"

	// Register all property types.
	_ "github.com/radityajay/notionctl/internal/property"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Pull current Notion database state back into config YAML",
	Long: `Fetches each managed database from Notion and regenerates the config YAML
to match the current remote state. Useful when databases have been edited
directly in Notion and you want to update your config to match.

After sync, review changes with 'git diff' before committing.`,
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
		imp := importer.New(client)

		result, err := imp.FromState(cfg, st)
		if err != nil {
			return err
		}

		// Write updated config
		data, err := importer.MarshalYAML(result.Config)
		if err != nil {
			return err
		}

		if err := os.WriteFile(configFile, data, 0o644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		// Update state
		if err := result.State.Save("."); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}

		fmt.Printf("✓ Synced %d database(s) from Notion → %s\n", len(result.Config.Databases), configFile)

		if warnings := importer.FormatWarnings(result.Warnings); warnings != "" {
			fmt.Print(warnings)
		}

		fmt.Println("\nReview changes with 'git diff' before committing.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
