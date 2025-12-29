package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeDockerFiles(cfg Config, backendDir string) error {
	projectRoot := cfg.ProjectName

	// This Dockerfile handles Go building, Bun building, and production targets
	dockerfile := `# Stage 1: Backend Builder
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ .
RUN go build -o main main.go

# Stage 2: Frontend Builder
FROM oven/bun:latest AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/bun.lockb* ./
RUN bun install
COPY frontend/ .
RUN bun run build

# Stage 3: Production (Backend API)
FROM alpine:latest AS prod
WORKDIR /app
COPY --from=backend-builder /app/main .
COPY --from=frontend-builder /app/dist ./dist
# Install certificates for HTTPS requests
RUN apk add --no-cache ca-certificates
EXPOSE 8080
CMD ["./main"]

# Stage 4: Frontend (Dev/Standalone)
FROM oven/bun:latest AS frontend
WORKDIR /app
COPY frontend/package.json frontend/bun.lockb* ./
RUN bun install
COPY frontend/ .
EXPOSE 5173
CMD ["bun", "run", "dev", "--host"]
`

	// 2. Create the Advanced docker-compose.yml
	compose := `services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
      target: prod
    restart: unless-stopped
    ports:
      - "${PORT}:${PORT}"
    env_file: .env
    environment:
      GOKOZYY_DB_HOST: psql_gokozyy
      GOKOZYY_DB_PORT: 5432
    depends_on:
      psql_gokozyy:
        condition: service_healthy
    networks:
      - gokozyy_network

  frontend:
    build:
      context: .
      dockerfile: Dockerfile
      target: frontend
    restart: unless-stopped
    ports:
      - "5173:5173"
    networks:
      - gokozyy_network

  psql_gokozyy:
    image: postgres:latest
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${GOKOZYY_DB_DATABASE}
      POSTGRES_USER: ${GOKOZYY_DB_USERNAME}
      POSTGRES_PASSWORD: ${GOKOZYY_DB_PW}
    ports:
      - "${GOKOZYY_DB_PORT}:5432"
    volumes:
      - psql_data_gokozyy:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "sh -c 'pg_isready -U ${GOKOZYY_DB_USERNAME} -d ${GOKOZYY_DB_DATABASE}'"]
      interval: 5s
      timeout: 5s
      retries: 3
      start_period: 15s
    networks:
      - gokozyy_network

volumes:
  psql_data_gokozyy:

networks:
  gokozyy_network:
`
	// Write Dockerfile
	if err := os.WriteFile(filepath.Join(projectRoot, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		return fmt.Errorf("writing Dockerfile: %w", err)
	}

	// Write docker-compose.yml
	if err := os.WriteFile(filepath.Join(projectRoot, "docker-compose.yml"), []byte(compose), 0o644); err != nil {
		return fmt.Errorf("writing docker-compose.yml: %w", err)
	}

	return nil
}
