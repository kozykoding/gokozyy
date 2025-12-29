package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Config is what you’ll pass from the TUI.
type Config struct {
	ProjectName string
	Framework   string // "std" | "chi" | "gin"
	DBDriver    string // "none" | "postgres" | "sqlite"
	Frontend    string // "vite-react-tailwind" | "vite-react-tailwind-shadcn"
	Runtime     string // "bun"
	UseDocker   bool   // whether to scaffold Docker for the DB
}

func generateFrontend(cfg Config) error {
	frontendDir := filepath.Join(cfg.ProjectName, "frontend")

	fmt.Printf("◦ Scaffolding frontend in %s...\n", frontendDir)
	fmt.Printf("  [gokozyy] cfg.Frontend = %q\n", cfg.Frontend)

	if err := runBunCreateVite(cfg.ProjectName, "frontend"); err != nil {
		return fmt.Errorf("bun create vite: %w", err)
	}

	if err := bunInstall(frontendDir); err != nil {
		return fmt.Errorf("bun install: %w", err)
	}

	// Tailwind v4 setup
	if err := setupTailwindV4(frontendDir); err != nil {
		return fmt.Errorf("tailwind v4 setup: %w", err)
	}

	// Only patch tsconfig and install shadcn when user selected that option
	if cfg.Frontend == "vite-react-tailwind-shadcn" {
		fmt.Println("  [gokozyy] calling setupShadcnManualV4...")
		if err := setupShadcnManualV4(frontendDir); err != nil {
			return fmt.Errorf("shadcn manual v4 setup: %w", err)
		}
	}

	return nil
}

func runGoModInit(dir, modulePath string) error {
	cmd := exec.Command("go", "mod", "init", modulePath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeStdMain(dir string) error {
	code := `package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ` + "`" + `{"status":"ok"}` + "`" + `)
	})

	addr := ":8080"
	log.Println("Starting standard-library server on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
`
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)
}

func generateBackend(cfg Config) error {
	backendDir := filepath.Join(cfg.ProjectName, "backend")

	// 1) Create backend directory
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		return fmt.Errorf("create backend dir: %w", err)
	}

	// 2) Initialize go module
	modulePath := fmt.Sprintf("github.com/you/%s/backend", cfg.ProjectName)
	if err := runGoModInit(backendDir, modulePath); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	// 3) main.go based on framework
	switch cfg.Framework {
	case "chi":
		if err := writeChiMain(backendDir); err != nil {
			return err
		}
	case "gin":
		if err := writeGinMain(backendDir); err != nil {
			return err
		}
	default:
		if err := writeStdMain(backendDir); err != nil {
			return err
		}
	}

	// 4) DB scaffolding (internal/database + driver imports)
	if err := setupDatabase(cfg, backendDir); err != nil {
		return fmt.Errorf("database setup: %w", err)
	}

	// 5) .env + .gitignore at project root
	if err := writeEnvFile(cfg); err != nil {
		return fmt.Errorf(".env: %w", err)
	}
	if err := writeGitignore(cfg); err != nil {
		return fmt.Errorf(".gitignore: %w", err)
	}

	// 6) makefile
	if err := writeMakefile(cfg); err != nil {
		return fmt.Errorf("makefile: %w", err)
	}

	// NEW: 7) Write Air config for hot reloading
	if err := writeAirConfig(cfg); err != nil {
		return fmt.Errorf("air config: %w", err)
	}

	// 8) Optional Docker files
	if cfg.UseDocker {
		if err := writeDockerFiles(cfg, backendDir); err != nil {
			return fmt.Errorf("docker: %w", err)
		}
	}
	return nil
}

// Generate is the main entry point called from cmd/create.go.
func Generate(cfg Config) error {
	// Top-level project directory (same as project name for now).
	if err := os.MkdirAll(cfg.ProjectName, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	// 1) Scaffold backend (TODO: your templates go here).
	if err := generateBackend(cfg); err != nil {
		return fmt.Errorf("backend: %w", err)
	}

	// 2) Scaffold frontend using Bun + Vite React.
	if err := generateFrontend(cfg); err != nil {
		return fmt.Errorf("frontend: %w", err)
	}

	return nil
}
