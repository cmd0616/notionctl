package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radityajay/notionctl/internal/config"

	// Register all property types.
	_ "github.com/radityajay/notionctl/internal/property"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate config file syntax and property types",
	Long: `Validates the notionctl YAML config without connecting to Notion.
Checks syntax, property types, relation references, formula/rollup config, and more.

Useful in CI to catch config errors before apply.
No NOTION_TOKEN required.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		fmt.Printf("✓ %s is valid (%d database(s), %d total properties)\n",
			configFile, len(cfg.Databases), countProperties(cfg))
		return nil
	},
}

func countProperties(cfg *config.Config) int {
	total := 0
	for _, db := range cfg.Databases {
		total += len(db.Properties)
	}
	return total
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
