package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

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
