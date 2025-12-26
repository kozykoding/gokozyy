package database

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
