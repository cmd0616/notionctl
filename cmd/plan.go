package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radityajayantara/notionctl/internal/config"
	"github.com/radityajayantara/notionctl/internal/engine"
	"github.com/radityajayantara/notionctl/internal/state"

	// Register all property types.
	_ "github.com/radityajayantara/notionctl/internal/property"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Preview changes between config and current Notion state",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		st, err := state.Load(".")
		if err != nil {
			return err
		}

		eng := engine.New(cfg, st, nil, ".")
		actions, err := eng.Plan()
		if err != nil {
			return err
		}

		fmt.Print(engine.FormatPlan(actions))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
