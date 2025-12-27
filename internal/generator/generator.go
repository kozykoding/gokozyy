package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	if err := runBunCreateVite(cfg.ProjectName, "frontend"); err != nil {
		return fmt.Errorf("bun create vite: %w", err)
	}

	if err := bunInstall(frontendDir); err != nil {
		return fmt.Errorf("bun install: %w", err)
	}

	// Tailwind setup (always, for both frontend options you defined)
	if err := setupTailwind(frontendDir); err != nil {
		return fmt.Errorf("tailwind setup: %w", err)
	}

	// shadcn/ui only if requested
	if cfg.Frontend == "vite-react-tailwind-shadcn" {
		if err := setupShadcn(frontendDir); err != nil {
			return fmt.Errorf("shadcn setup: %w", err)
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

func writeChiMain(dir string) error {
	code := `package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(` + "`" + `{"status":"ok"}` + "`" + `))
	})

	addr := ":8080"
	log.Println("Starting chi server on", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
`
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)
}

func writeGinMain(dir string) error {
	code := `package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := ":8080"
	log.Println("Starting gin server on", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
`
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644)
}

func setupTailwind(frontendDir string) error {
	fmt.Println("◦ Installing Tailwind CSS (Vite plugin)...")

	// Install tailwindcss, the Vite plugin, and @types/node
	cmd := exec.Command("bun", "add", "-D",
		"tailwindcss",
		"@tailwindcss/vite",
		"@types/node",
	)
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Tailwind config in TypeScript
	twConfig := `import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {},
  },
  plugins: [],
};

export default config;
`
	if err := os.WriteFile(
		filepath.Join(frontendDir, "tailwind.config.ts"),
		[]byte(twConfig),
		0o644,
	); err != nil {
		return err
	}

	// Tailwind directives in index.css
	indexCSS := `@tailwind base;
@tailwind components;
@tailwind utilities;
`
	if err := os.WriteFile(
		filepath.Join(frontendDir, "src", "index.css"),
		[]byte(indexCSS),
		0o644,
	); err != nil {
		return err
	}

	// Ensure main.tsx imports index.css
	mainPath := filepath.Join(frontendDir, "src", "main.tsx")
	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		return err
	}
	if !strings.Contains(string(mainBytes), `./index.css`) {
		mainBytes = append(
			[]byte(`import "./index.css";`+"\n"),
			mainBytes...,
		)
		if err := os.WriteFile(mainPath, mainBytes, 0o644); err != nil {
			return err
		}
	}

	// Overwrite vite.config.ts to match the Tailwind + alias pattern
	if err := writeViteConfigWithTailwind(frontendDir); err != nil {
		return err
	}

	return nil
}

func patchViteConfigForTailwind(frontendDir string) error {
	vitePath := filepath.Join(frontendDir, "vite.config.ts")
	data, err := os.ReadFile(vitePath)
	if err != nil {
		return err
	}

	content := string(data)

	if strings.Contains(content, "@tailwindcss/vite") {
		return nil
	}

	// 1) Ensure import
	lines := strings.Split(content, "\n")
	importInserted := false
	for i, line := range lines {
		if strings.HasPrefix(line, "import react") && !importInserted {
			lines[i] = line + "\nimport tailwindcss from \"@tailwindcss/vite\";"
			importInserted = true
			break
		}
	}

	// 2) Ensure plugins array contains tailwindcss()
	for i, line := range lines {
		if strings.Contains(line, "plugins: [react(") || strings.Contains(line, "plugins: [react(") {
			// turn into plugins: [react(), tailwindcss()],
			if !strings.Contains(line, "tailwindcss(") {
				line = strings.Replace(line, "react()", "react(), tailwindcss()", 1)
				lines[i] = line
			}
			break
		}
	}

	newContent := strings.Join(lines, "\n")
	return os.WriteFile(vitePath, []byte(newContent), 0o644)
}

func ensureTsconfigAlias(frontendDir string) error {
	if err := patchTsconfig(filepath.Join(frontendDir, "tsconfig.json")); err != nil {
		return err
	}
	if err := patchTsconfig(filepath.Join(frontendDir, "tsconfig.app.json")); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func patchTsconfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// "compilerOptions": { "baseUrl": ".", "paths": { "@/*": ["./src/*"] } }

	var ts map[string]any
	if err := json.Unmarshal(data, &ts); err != nil {
		return err
	}

	compiler, _ := ts["compilerOptions"].(map[string]any)
	if compiler == nil {
		compiler = map[string]any{}
	}

	if _, ok := compiler["baseUrl"]; !ok {
		compiler["baseUrl"] = "."
	}

	paths, _ := compiler["paths"].(map[string]any)
	if paths == nil {
		paths = map[string]any{}
	}

	if _, ok := paths["@/*"]; !ok {
		paths["@/*"] = []any{"./src/*"}
	}

	compiler["paths"] = paths
	ts["compilerOptions"] = compiler

	out, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0o644)
}

func writeViteConfigWithTailwind(frontendDir string) error {
	viteContent := `import path from "path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
`
	return os.WriteFile(
		filepath.Join(frontendDir, "vite.config.ts"),
		[]byte(viteContent),
		0o644,
	)
}

func setupShadcn(frontendDir string) error {
	fmt.Println("◦ Setting up shadcn/ui with default options...")

	// 1) Ensure tsconfig alias (@/* -> ./src/*) in both tsconfig files
	if err := ensureTsconfigAlias(frontendDir); err != nil {
		return fmt.Errorf("tsconfig alias: %w", err)
	}

	// 2) Initialize shadcn/ui (auto-yes to questions)
	fmt.Println("  - Running: bunx --bun shadcn@latest init (auto-yes)")
	initCmd := exec.Command(
		"bash",
		"-lc",
		"yes | bunx --bun shadcn@latest init",
	)
	initCmd.Dir = frontendDir
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("shadcn init: %w", err)
	}

	// 3) Add button component (no yes pipe unless we see it needed)
	fmt.Println("  - Running: bunx --bun shadcn@latest add button")
	addCmd := exec.Command(
		"bash",
		"-lc",
		"bunx --bun shadcn@latest add button",
	)
	addCmd.Dir = frontendDir
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("shadcn add button: %w", err)
	}

	// 4) Verify the button file exists; fail loudly if it doesn't.
	buttonPath := filepath.Join(frontendDir, "components", "ui", "button.tsx")
	if _, err := os.Stat(buttonPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("shadcn add button: expected %s but it was not created", buttonPath)
		}
		return fmt.Errorf("checking shadcn button file: %w", err)
	}

	fmt.Println("◦ shadcn/ui initialized and button component added.")
	return nil
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

	// 6) Optional Docker files
	if cfg.UseDocker {
		if err := writeDockerFiles(cfg, backendDir); err != nil {
			return fmt.Errorf("docker: %w", err)
		}
	}

	return nil
}

func setupDatabase(cfg Config, backendDir string) error {
	if cfg.DBDriver == "none" {
		return nil
	}

	dbDir := filepath.Join(backendDir, "internal", "database")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("create database dir: %w", err)
	}

	switch cfg.DBDriver {
	case "postgres":
		return writePostgresDatabase(dbDir)
	case "sqlite":
		return writeSQLiteDatabase(dbDir)
	default:
		return nil
	}
}

func writePostgresDatabase(dir string) error {
	code := `package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres() (*sql.DB, error) {
	host := os.Getenv("GOKOZYY_DB_HOST")
	port := os.Getenv("GOKOZYY_DB_PORT")
	user := os.Getenv("GOKOZYY_DB_USERNAME")
	pw   := os.Getenv("GOKOZYY_DB_PW")
	db   := os.Getenv("GOKOZYY_DB_DATABASE")
	ssl  := "disable"

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, pw, host, port, db, ssl,
	)

	return sql.Open("pgx", dsn)
}
`
	return os.WriteFile(filepath.Join(dir, "database.go"), []byte(code), 0o644)
}

func writeSQLiteDatabase(dir string) error {
	code := `package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func NewSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path)
}
`
	return os.WriteFile(filepath.Join(dir, "database.go"), []byte(code), 0o644)
}

func writeEnvFile(cfg Config) error {
	envPath := filepath.Join(cfg.ProjectName, ".env")

	content := `PORT=42069
APP_ENV=local
GOKOZYY_DB_HOST=localhost
GOKOZYY_DB_PORT=5432
GOKOZYY_DB_DATABASE=gokozyy
GOKOZYY_DB_USERNAME=sammy
GOKOZYY_DB_PW=thisismypassword
GOKOZYY_DB_SCHEMA=public
`

	return os.WriteFile(envPath, []byte(content), 0o600)
}

func writeGitignore(cfg Config) error {
	path := filepath.Join(cfg.ProjectName, ".gitignore")

	content := `.env
# Go
bin/
*.exe
*.test
*.out

# Node/Bun/Vite
node_modules/
dist/
.vite/

# IDE/editor
.vscode/
.idea/
.DS_Store
`

	return os.WriteFile(path, []byte(content), 0o644)
}

func writeDockerFiles(cfg Config, backendDir string) error {
	projectRoot := cfg.ProjectName

	// docker-compose with Postgres if DBDriver == postgres
	if cfg.DBDriver == "postgres" {
		compose := `version: "3.9"

services:
  db:
    image: postgres:15
    container_name: ` + cfg.ProjectName + `_db
    restart: unless-stopped
    environment:
      POSTGRES_DB: gokozyy
      POSTGRES_USER: sammy
      POSTGRES_PASSWORD: thisismypassword
    ports:
      - "5432:5432"
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
`
		if err := os.WriteFile(
			filepath.Join(projectRoot, "docker-compose.yml"),
			[]byte(compose),
			0o644,
		); err != nil {
			return err
		}
	}

	// Simple Dockerfile for backend service
	dockerfile := `FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY backend ./backend
WORKDIR /app/backend
RUN go build -o server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/backend/server /app/server
CMD ["./server"]
`
	if err := os.WriteFile(
		filepath.Join(projectRoot, "Dockerfile"),
		[]byte(dockerfile),
		0o644,
	); err != nil {
		return err
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
