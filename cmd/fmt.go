package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radityajay/notionctl/internal/config"

	// Register all property types.
	_ "github.com/radityajay/notionctl/internal/property"
)

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Format config file with consistent style",
	Long: `Formats the notionctl YAML config with consistent ordering and style.
Databases are sorted by name. Properties are sorted with title first, then alphabetically.

Modifies the file in place. Use 'git diff' to review changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		changed, err := config.FormatFile(configFile)
		if err != nil {
			return err
		}

		if changed {
			fmt.Printf("✓ Formatted %s\n", configFile)
		} else {
			fmt.Printf("✓ %s is already formatted\n", configFile)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fmtCmd)
}
