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
	Long: `A quick way to start up / bootstrap a project with
separate front and backends. 

Run the gokozyy create command inside the directory where you want 
your new project folder to be created.`,
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

		cfg := generator.Config{
			ProjectName: res.ProjectName,
			Framework:   res.Framework,
			DBDriver:    res.DBDriver,
			Frontend:    res.Frontend,
			Runtime:     res.Runtime,
			UseDocker:   res.UseDocker,
		}

		if err := generator.Generate(cfg); err != nil {
			return err
		}

		fmt.Println()
		fmt.Printf("✅ Project %q created successfully!\n", cfg.ProjectName)
		fmt.Println()
		fmt.Println("🚀 Next steps to start nerding out:")
		fmt.Printf("  1. cd %s && nvim .\n", cfg.ProjectName)

		// Only show docker instruction if they chose Docker + Postgres
		if cfg.UseDocker && cfg.DBDriver == "postgres" {
			fmt.Println("  2. make docker-run         # Start your Postgres database")
		}

		fmt.Println("  3. make watch              # Start backend with hot-reload (Air)")
		fmt.Println("  4. cd frontend && bun dev   # Start React frontend environment")
		fmt.Println()
		fmt.Println("Happy coding!")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
