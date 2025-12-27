package generator

import (
	"os"
	"path/filepath"
)

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
