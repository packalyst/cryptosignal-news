# CryptoSignal News Backend Makefile
# ===================================

# Variables
APP_NAME := cryptosignal-news
API_BINARY := cmd/api/main.go
FETCHER_BINARY := cmd/fetcher/main.go
BUILD_DIR := ./bin
DOCKER_IMAGE := $(APP_NAME)
GO := go

# Environment
ENV ?= development
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/cryptosignal?sslmode=disable
REDIS_URL ?= redis://localhost:6379/0

# Build flags
LDFLAGS := -ldflags "-s -w"

.PHONY: all build build-api build-fetcher run run-api run-fetcher test test-coverage lint fmt vet clean deps tidy migrate migrate-down docker-build docker-run docker-stop help

# Default target
all: build

## Build targets
build: build-api build-fetcher
	@echo "Build complete"

build-api:
	@echo "Building API server..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/api $(API_BINARY)

build-fetcher:
	@echo "Building fetcher..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/fetcher $(FETCHER_BINARY)

## Run targets
run: run-api

run-api:
	@echo "Starting API server..."
	$(GO) run $(API_BINARY)

run-fetcher:
	@echo "Starting fetcher..."
	$(GO) run $(FETCHER_BINARY)

## Test targets
test:
	@echo "Running tests..."
	$(GO) test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short:
	@echo "Running short tests..."
	$(GO) test -v -short ./...

## Code quality targets
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@which goimports > /dev/null && goimports -w . || true

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

check: fmt vet lint test
	@echo "All checks passed!"

## Dependency management
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify

## Database migrations
migrate:
	@echo "Running migrations..."
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "DATABASE_URL is not set"; \
		exit 1; \
	fi
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql "$(DATABASE_URL)" -f $$f; \
	done
	@echo "Migrations complete"

migrate-down:
	@echo "Rolling back migrations..."
	@echo "Not implemented - please write rollback migrations manually"

migrate-status:
	@echo "Checking migration status..."
	@psql "$(DATABASE_URL)" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';"

## Docker targets
docker-build:
	@echo "Building Docker images..."
	docker build -t $(DOCKER_IMAGE)-api -f Dockerfile .
	docker build -t $(DOCKER_IMAGE)-fetcher -f Dockerfile.fetcher .

docker-build-api:
	@echo "Building API Docker image..."
	docker build -t $(DOCKER_IMAGE)-api -f Dockerfile .

docker-build-fetcher:
	@echo "Building fetcher Docker image..."
	docker build -t $(DOCKER_IMAGE)-fetcher -f Dockerfile.fetcher .

docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker services..."
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v --remove-orphans
	docker rmi $(DOCKER_IMAGE)-api $(DOCKER_IMAGE)-fetcher 2>/dev/null || true

## Development helpers
dev: deps
	@echo "Starting development server with hot reload..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

seed:
	@echo "Seeding database..."
	$(GO) run scripts/seed.go

generate:
	@echo "Running go generate..."
	$(GO) generate ./...

## Cleanup
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean -cache

## Help
help:
	@echo "CryptoSignal News Backend - Available Commands"
	@echo ""
	@echo "Build:"
	@echo "  make build           Build all binaries"
	@echo "  make build-api       Build API server"
	@echo "  make build-fetcher   Build fetcher service"
	@echo ""
	@echo "Run:"
	@echo "  make run             Run API server"
	@echo "  make run-api         Run API server"
	@echo "  make run-fetcher     Run fetcher service"
	@echo ""
	@echo "Test:"
	@echo "  make test            Run all tests"
	@echo "  make test-coverage   Run tests with coverage report"
	@echo "  make test-short      Run short tests only"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint            Run linter"
	@echo "  make fmt             Format code"
	@echo "  make vet             Run go vet"
	@echo "  make check           Run all checks (fmt, vet, lint, test)"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps            Download dependencies"
	@echo "  make tidy            Tidy go.mod"
	@echo "  make verify          Verify dependencies"
	@echo ""
	@echo "Database:"
	@echo "  make migrate         Run database migrations"
	@echo "  make migrate-status  Check migration status"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build    Build Docker images"
	@echo "  make docker-run      Start services with Docker Compose"
	@echo "  make docker-stop     Stop Docker services"
	@echo "  make docker-logs     View Docker logs"
	@echo "  make docker-clean    Clean Docker resources"
	@echo ""
	@echo "Development:"
	@echo "  make dev             Start development server with hot reload"
	@echo "  make seed            Seed the database"
	@echo "  make clean           Clean build artifacts"
