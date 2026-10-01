package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const initTemplate = `# notionctl configuration
# Docs: https://github.com/radityajayantara/notionctl
version: "1"

databases:
  - name: Projects
    parent_page_id: "YOUR_NOTION_PAGE_ID_HERE"
    properties:
      Name:
        type: title
      Description:
        type: rich_text
      Status:
        type: select
        options:
          - name: Not Started
            color: red
          - name: In Progress
            color: yellow
          - name: Done
            color: green
      tasks:
        type: relation
        relation: Tasks

  - name: Tasks
    parent_page_id: "YOUR_NOTION_PAGE_ID_HERE"
    properties:
      Name:
        type: title
      Priority:
        type: number
        format: number
      Tags:
        type: multi_select
        options:
          - name: bug
            color: red
          - name: feature
            color: blue
          - name: docs
            color: gray
      project:
        type: relation
        relation: Projects
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter notionctl.yaml config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(configFile); err == nil {
			return fmt.Errorf("%s already exists — remove it first or use a different name with -c", configFile)
		}

		if err := os.WriteFile(configFile, []byte(initTemplate), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", configFile, err)
		}

		fmt.Printf("Created %s\n", configFile)
		fmt.Println("Next steps:")
		fmt.Println("  1. Replace YOUR_NOTION_PAGE_ID_HERE with your Notion page ID")
		fmt.Println("  2. Set NOTION_TOKEN environment variable")
		fmt.Println("  3. Run: notionctl plan")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
