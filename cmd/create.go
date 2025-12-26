package cmd

/*
Copyright © 2025 SAMMY SAMMY@KOZYKODING.COM
*/

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kozykoding/gokozyy/internal/generator"
	"github.com/kozykoding/gokozyy/internal/ui"
	"github.com/spf13/cobra"
)

// flags for non-interactive mode
var (
	flagName      string
	flagFramework string
	flagDB        string
	flagNoTUI     bool
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Go + Vite project with Bun",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		m := ui.NewWizardModel()
		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		wm := finalModel.(ui.WizardModel)
		res := wm.Result()
		if !res.Confirmed {
			fmt.Println("Cancelled.")
			return nil
		}

		// Uncomment for prod
		cfg := generator.Config{
			ProjectName: res.ProjectName,
			Framework:   res.Framework,
			DBDriver:    res.DBDriver,
			Frontend:    res.Frontend,
			Runtime:     res.Runtime,
			UseDocker:   res.UseDocker,
		}

		// for testing
		// simple demo: create folder
		//	if err := os.MkdirAll(res.ProjectName, 0o755); err != nil {
		//		return err
		//	}

		//	return nil

		// for production to create
		return generator.Generate(cfg)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
