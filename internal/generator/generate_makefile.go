package generator

import (
	"os"
	"path/filepath"
)

func writeMakefile(cfg Config) error {
	projectRoot := cfg.ProjectName

	makefileContent := `# Simple Makefile for Gokozyy project

# Build the application
all: build test

build:
	@echo "Building..."
	@go build -o main backend/main.go

# Run the application
run:
	@go run backend/main.go

# Create DB container
docker-run:
	@if docker compose up psql_gokozyy -d 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up psql_gokozyy -d; \
	fi

# Shutdown DB container
docker-down:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload (Go)
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
            else \
                echo "Skipping air install."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch docker-run docker-down
`
	return os.WriteFile(filepath.Join(projectRoot, "Makefile"), []byte(makefileContent), 0o644)
}
