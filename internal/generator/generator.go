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
	frontendDir := cfg.ProjectName + "/frontend"

	if err := runBunCreateVite(cfg.ProjectName, "frontend"); err != nil {
		return fmt.Errorf("bun create vite: %w", err)
	}

	if err := bunInstall(frontendDir); err != nil {
		return fmt.Errorf("bun install: %w", err)
	}

	// later: setupTailwind / setupShadcn based on cfg.Frontend
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
