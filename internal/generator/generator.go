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

func setupTailwindV4(frontendDir string) error {
	fmt.Println("◦ Installing Tailwind CSS v4 (Vite plugin)...")

	// Install tailwindcss and the Vite plugin (and @types/node for TS tooling)
	cmd := exec.Command("bun", "add", "-D",
		"tailwindcss",
		"@tailwindcss/vite",
		"@types/node",
	)
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun add tailwind v4 deps: %w", err)
	}

	// Tailwind v4-style config in TypeScript
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
		return fmt.Errorf("write tailwind.config.ts: %w", err)
	}

	// Tailwind v4 CSS entry: @import "tailwindcss";
	indexCSS := `@import "tailwindcss";
`
	if err := os.WriteFile(
		filepath.Join(frontendDir, "src", "index.css"),
		[]byte(indexCSS),
		0o644,
	); err != nil {
		return fmt.Errorf("write src/index.css: %w", err)
	}

	// Ensure main.tsx imports index.css
	mainPath := filepath.Join(frontendDir, "src", "main.tsx")
	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("read main.tsx: %w", err)
	}
	if !strings.Contains(string(mainBytes), `./index.css`) {
		mainBytes = append(
			[]byte(`import "./index.css";`+"\n"),
			mainBytes...,
		)
		if err := os.WriteFile(mainPath, mainBytes, 0o644); err != nil {
			return fmt.Errorf("write main.tsx: %w", err)
		}
	}

	// Vite config with Tailwind plugin + @ alias
	if err := writeViteConfigWithTailwindV4(frontendDir); err != nil {
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

func writeViteConfigWithTailwindV4(frontendDir string) error {
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

func ensureRootTsconfigAlias(frontendDir string) error {
	tsconfigPath := filepath.Join(frontendDir, "tsconfig.json")

	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return fmt.Errorf("read tsconfig.json: %w", err)
	}

	var ts map[string]any
	if err := json.Unmarshal(data, &ts); err != nil {
		return fmt.Errorf("parse tsconfig.json: %w", err)
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
		return fmt.Errorf("marshal tsconfig.json: %w", err)
	}

	if err := os.WriteFile(tsconfigPath, out, 0o644); err != nil {
		return fmt.Errorf("write tsconfig.json: %w", err)
	}

	return nil
}

func patchRootTsconfig(frontendDir string) error {
	tsconfigPath := filepath.Join(frontendDir, "tsconfig.json")

	// No parsing, no "invalid character" errors.
	content := `{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" },
    { "path": "./tsconfig.node.json" }
  ],
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
`
	return os.WriteFile(tsconfigPath, []byte(content), 0o644)
}

func patchAppTsconfig(frontendDir string) error {
	appPath := filepath.Join(frontendDir, "tsconfig.app.json")

	// This preserves the default Vite settings but adds shadcn requirements.
	content := `{
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.app.tsbuildinfo",
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,

    /* Bundler mode */
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "verbatimModuleSyntax": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",

    /* Linting */
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true,

    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src"]
}
`
	return os.WriteFile(appPath, []byte(content), 0o644)
}

func setupShadcnManualV4(frontendDir string) error {
	fmt.Println("◦ Setting up shadcn/ui (manual, Tailwind v4)...")

	// 1) Ensure tsconfig.json and tsconfig.app.json have the alias shadcn expects
	if err := patchRootTsconfig(frontendDir); err != nil {
		return fmt.Errorf("patch root tsconfig: %w", err)
	}
	if err := patchAppTsconfig(frontendDir); err != nil {
		return fmt.Errorf("patch app tsconfig: %w", err)
	}

	// 2) Add shadcn-related deps
	cmd := exec.Command("bun", "add",
		"lucide-react",
		"class-variance-authority",
		"clsx",
		"tailwind-merge",
		"tailwindcss-animate",
	)
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun add shadcn deps: %w", err)
	}

	// 3) components.json (points to tailwind.config.ts and src/index.css)
	componentsJSON := `{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.ts",
    "css": "src/index.css",
    "baseColor": "neutral"
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils"
  }
}
`
	if err := os.WriteFile(
		filepath.Join(frontendDir, "components.json"),
		[]byte(componentsJSON),
		0o644,
	); err != nil {
		return fmt.Errorf("write components.json: %w", err)
	}

	// 4) src/components/ui/button.tsx
	uiDir := filepath.Join(frontendDir, "src", "components", "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		return fmt.Errorf("create src/components/ui: %w", err)
	}

	button := `import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none",
  {
    variants: {
      variant: {
        default: "bg-neutral-900 text-neutral-50 hover:bg-neutral-800",
        outline: "border border-neutral-200 hover:bg-neutral-100",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 px-3",
        lg: "h-10 px-8",
      }
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";

export { Button, buttonVariants };
`
	if err := os.WriteFile(
		filepath.Join(uiDir, "button.tsx"),
		[]byte(button),
		0o644,
	); err != nil {
		return fmt.Errorf("write button.tsx: %w", err)
	}

	// 5) src/lib/utils.ts for cn()
	libDir := filepath.Join(frontendDir, "src", "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return fmt.Errorf("create src/lib: %w", err)
	}

	utils := `import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
`
	if err := os.WriteFile(
		filepath.Join(libDir, "utils.ts"),
		[]byte(utils),
		0o644,
	); err != nil {
		return fmt.Errorf("write src/lib/utils.ts: %w", err)
	}

	fmt.Println("◦ shadcn/ui (manual v4) installed: components.json, src/components/ui/button.tsx, src/lib/utils.ts")
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
