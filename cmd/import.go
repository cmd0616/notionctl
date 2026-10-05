package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/radityajay/notionctl/internal/importer"
	"github.com/radityajay/notionctl/internal/notion"

	// Register all property types (needed for config validation).
	_ "github.com/radityajay/notionctl/internal/property"
)

var (
	importPageID string
	importOutput string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import existing Notion databases into a notionctl config",
	Long: `Import scans a Notion page for databases and generates a notionctl.yaml
config file plus state, so you can manage existing databases declaratively.

  notionctl import --page-id <PAGE_ID>
  notionctl import --page-id <PAGE_ID> -o my-config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if importPageID == "" {
			return fmt.Errorf("--page-id is required\nUsage: notionctl import --page-id <PAGE_ID>")
		}

		token := os.Getenv("NOTION_TOKEN")
		if token == "" {
			return fmt.Errorf("NOTION_TOKEN environment variable is required\nGet one at https://www.notion.so/my-integrations")
		}

		// Check output file doesn't exist
		if _, err := os.Stat(importOutput); err == nil {
			return fmt.Errorf("output file %q already exists — use -o to specify a different path", importOutput)
		}

		client := notion.NewClient(token)
		imp := importer.New(client)

		fmt.Printf("Scanning page %s for databases...\n", importPageID)

		result, err := imp.FromPage(importPageID)
		if err != nil {
			return err
		}

		fmt.Printf("Found %d database(s)\n", len(result.Config.Databases))
		for _, db := range result.Config.Databases {
			fmt.Printf("  • %s (%d properties)\n", db.Name, len(db.Properties))
		}

		// Write YAML
		yamlData, err := importer.MarshalYAML(result.Config)
		if err != nil {
			return err
		}

		if err := os.WriteFile(importOutput, yamlData, 0o644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		fmt.Printf("\n✓ Config written to %s\n", importOutput)

		// Write state
		if err := result.State.Save("."); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}
		fmt.Println("✓ State saved to .notionctl/state.json")

		// Show warnings
		if w := importer.FormatWarnings(result.Warnings); w != "" {
			fmt.Print(w)
		}

		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Review %s\n", importOutput)
		fmt.Println("  2. Run: notionctl plan")
		return nil
	},
}

func init() {
	importCmd.Flags().StringVar(&importPageID, "page-id", "", "Notion page ID containing databases")
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "notionctl.yaml", "output YAML file path")
	rootCmd.AddCommand(importCmd)
}
