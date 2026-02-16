.PHONY: build build-fast build-ui run test clean docker-build docker-run lint fmt deps ui

# Binary name
BINARY=gprint
UI_BINARY=gprint-ui
MAIN_PATH=./cmd/server
UI_PATH=./cmd/ui

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILDFLAGS=-trimpath $(LDFLAGS)

# Default target
all: deps lint test build

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build the application
build:
	@mkdir -p bin
	$(GOBUILD) $(BUILDFLAGS) -o bin/$(BINARY) $(MAIN_PATH)

# Fast build for development (skip optimizations)
build-fast:
	@mkdir -p bin
	$(GOBUILD) -o bin/$(BINARY) $(MAIN_PATH)

# Build the UI application
build-ui:
	@mkdir -p bin
	$(GOBUILD) $(BUILDFLAGS) -o bin/$(UI_BINARY) $(UI_PATH)

# Build for Linux (for Docker)
build-linux:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILDFLAGS) -o bin/$(BINARY)-linux $(MAIN_PATH)

# Load .env file and run the application
run:
	@if [ -f .env ]; then \
		export $$(grep -v '^#' .env | xargs) && \
		DYLD_LIBRARY_PATH=$(CURDIR)/lib TNS_ADMIN=$(CURDIR)/wallet $(GORUN) $(MAIN_PATH); \
	else \
		DYLD_LIBRARY_PATH=$(CURDIR)/lib TNS_ADMIN=$(CURDIR)/wallet $(GORUN) $(MAIN_PATH); \
	fi

# Run the UI application
ui:
	$(GORUN) $(UI_PATH)

# Run the UI SSH server
ui-ssh:
	@if [ -f .env ]; then \
		export $$(grep -v '^#' .env | xargs) && \
		$(GORUN) $(UI_PATH) serve; \
	else \
		$(GORUN) $(UI_PATH) serve; \
	fi

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@if [ -f .env ]; then \
		export $$(grep -v '^#' .env | xargs) && \
		DYLD_LIBRARY_PATH=$(CURDIR)/lib TNS_ADMIN=$(CURDIR)/wallet air; \
	else \
		DYLD_LIBRARY_PATH=$(CURDIR)/lib TNS_ADMIN=$(CURDIR)/wallet air; \
	fi

# Run tests
test:
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	$(GOFMT) -s -w .

# Lint code (requires golangci-lint)
lint:
	$(GOLINT) run ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker build
docker-build:
	docker build -t gprint:latest .

# Docker run
docker-run:
	docker run -p 8080:8080 --env-file .env gprint:latest

# Docker compose up
docker-up:
	docker-compose up -d

# Docker compose down
docker-down:
	docker-compose down

# Generate mocks (requires mockgen)
mocks:
	go generate ./...

# Database migration (example - adjust for your migration tool)
migrate-up:
	@echo "Run migrations manually using Oracle SQL*Plus or your preferred tool"
	@echo "Migration file: migrations/001_initial_schema.sql"

# Show help
help:
	@echo "Available targets:"
	@echo "  all           - Run deps, lint, test, and build"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  build         - Build the application"
	@echo "  build-ui      - Build the UI application"
	@echo "  build-linux   - Build for Linux (Docker)"
	@echo "  run           - Run the application"
	@echo "  ui            - Run the UI application"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docker-up     - Start with docker-compose"
	@echo "  docker-down   - Stop docker-compose"
	@echo "  help          - Show this help"
