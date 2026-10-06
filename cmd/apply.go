package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/radityajay/notionctl/internal/config"
	"github.com/radityajay/notionctl/internal/engine"
	"github.com/radityajay/notionctl/internal/notion"
	"github.com/radityajay/notionctl/internal/state"

	// Register all property types.
	_ "github.com/radityajay/notionctl/internal/property"
)

var autoApprove bool

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Create or update Notion databases to match config",
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

		// Set up destroy confirmation
		eng.SetConfirmDestroy(func(name, id string) bool {
			if autoApprove {
				return true
			}
			fmt.Printf("⚠ Destroy database %q (ID: %s)? This will archive it in Notion. [y/N] ", name, id)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			return answer == "y" || answer == "yes"
		})

		// Show plan first
		actions, err := eng.Plan()
		if err != nil {
			return err
		}
		if len(actions) == 0 {
			fmt.Println("No changes. Your Notion workspace matches the config.")
			return nil
		}

		fmt.Print(engine.FormatPlan(actions))
		fmt.Println("Applying...")
		fmt.Println()

		results, err := eng.Apply()
		if err != nil {
			return err
		}

		for _, r := range results {
			icon := "✓"
			if r.Type == "destroy" {
				icon = "✗"
			}
			fmt.Printf("%s %s %q\n", icon, r.Type, r.DatabaseName)
			for _, d := range r.Details {
				fmt.Printf("  %s\n", d)
			}
		}

		fmt.Println("\nDone. State saved to .notionctl/state.json")
		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "skip confirmation prompts for destructive actions")
	rootCmd.AddCommand(applyCmd)
}
