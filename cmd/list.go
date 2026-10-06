package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/radityajay/notionctl/internal/state"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed databases and their Notion IDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load(".")
		if err != nil {
			return err
		}

		if len(st.Databases) == 0 {
			fmt.Println("No managed databases. Run 'notionctl apply' first.")
			return nil
		}

		// Sort by name
		names := make([]string, 0, len(st.Databases))
		for name := range st.Databases {
			names = append(names, name)
		}
		sort.Strings(names)

		// Find max name length for alignment
		maxLen := 0
		for _, name := range names {
			if len(name) > maxLen {
				maxLen = len(name)
			}
		}

		fmt.Printf("Managed databases (%d):\n\n", len(names))
		for _, name := range names {
			db := st.Databases[name]
			propCount := len(db.Properties)
			fmt.Printf("  %-*s  %s  (%d properties)\n", maxLen, name, db.ID, propCount)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
