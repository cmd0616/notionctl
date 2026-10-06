package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const defaultTemplate = `# notionctl configuration
# Docs: https://github.com/radityajay/notionctl
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
        type: status
        options:
          - name: Not Started
            color: red
          - name: In Progress
            color: yellow
          - name: Done
            color: green
        groups:
          - name: To-do
            option_ids:
              - Not Started
          - name: In progress
            option_ids:
              - In Progress
          - name: Complete
            option_ids:
              - Done
      Start Date:
        type: date
      Website:
        type: url
      Updated:
        type: last_edited_time
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
      Done:
        type: checkbox
      Due Date:
        type: date
      Tags:
        type: multi_select
        options:
          - name: bug
            color: red
          - name: feature
            color: blue
          - name: docs
            color: gray
      Assignee Email:
        type: email
      project:
        type: relation
        relation: Projects
`

type templateChoice struct {
	name string
	desc string
	file string // empty = default template
}

var templates = []templateChoice{
	{name: "blank", desc: "Minimal starter (Projects + Tasks)"},
	{name: "crm", desc: "Sales pipeline (Contacts, Deals)", file: "crm.yaml"},
	{name: "inventory", desc: "Stock management (Products, Suppliers)", file: "inventory.yaml"},
	{name: "project-tracker", desc: "Project management (Projects, Milestones, Tasks)", file: "project-tracker.yaml"},
	{name: "bug-tracker", desc: "Issue tracking (Bugs, Components, Releases)", file: "bug-tracker.yaml"},
}

var templateFlag string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter notionctl.yaml config",
	Long: `Generate a starter YAML config file. Choose from built-in templates
or start with a minimal blank config.

Interactive mode: run without --template to choose interactively.
CI mode: use --template to skip prompts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(configFile); err == nil {
			return fmt.Errorf("%s already exists — remove it first or use a different name with -c", configFile)
		}

		selected := templateFlag
		if selected == "" {
			var err error
			selected, err = promptTemplate()
			if err != nil {
				selected = "blank"
			}
		}

		content, err := getTemplateContent(selected)
		if err != nil {
			return err
		}

		if err := os.WriteFile(configFile, content, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", configFile, err)
		}

		fmt.Printf("Created %s", configFile)
		if selected != "blank" {
			fmt.Printf(" (from %s template)", selected)
		}
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Replace YOUR_PAGE_ID / YOUR_NOTION_PAGE_ID_HERE with your Notion page ID")
		fmt.Println("  2. Set NOTION_TOKEN environment variable")
		fmt.Println("  3. Run: notionctl plan")
		return nil
	},
}

func promptTemplate() (string, error) {
	fmt.Println("Choose a template:")
	fmt.Println()
	for i, t := range templates {
		fmt.Printf("  %d) %-18s %s\n", i+1, t.name, t.desc)
	}
	fmt.Println()
	fmt.Print("Enter number [1]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "blank", err
	}
	answer = strings.TrimSpace(answer)

	if answer == "" {
		return "blank", nil
	}

	num, err := strconv.Atoi(answer)
	if err != nil || num < 1 || num > len(templates) {
		return "blank", fmt.Errorf("invalid choice: %s", answer)
	}

	return templates[num-1].name, nil
}

func getTemplateContent(name string) ([]byte, error) {
	if name == "blank" {
		return []byte(defaultTemplate), nil
	}

	for _, t := range templates {
		if t.name == name {
			// Read from templates/ directory relative to binary
			data, err := os.ReadFile(filepath.Join("templates", t.file))
			if err != nil {
				// Fallback: try embedded or return clear error
				return nil, fmt.Errorf("template %q not found — make sure templates/ directory is present, or use --template blank", name)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("unknown template: %q (available: blank, crm, inventory, project-tracker, bug-tracker)", name)
}

func init() {
	initCmd.Flags().StringVar(&templateFlag, "template", "", "template to use (blank, crm, inventory, project-tracker, bug-tracker)")
	rootCmd.AddCommand(initCmd)
}
