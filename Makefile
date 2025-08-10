.PHONY: help build run test clean docker-build docker-run docker-stop docker-logs

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the Go application"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-stop  - Stop Docker services"
	@echo "  docker-logs  - View Docker logs"
	@echo "  setup        - Initial setup"
	@echo "  migrate      - Run database migrations"

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run the application locally
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Build Docker image
docker-build:
	docker-compose build

# Run with Docker Compose
docker-run:
	docker-compose up -d

# Run with Docker Compose and rebuild
docker-run-build:
	docker-compose up -d --build

# Stop Docker services
docker-stop:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f

# View logs for specific service
docker-logs-app:
	docker-compose logs -f app

# Initial setup
setup:
	@echo "Setting up the eSIM platform..."
	@if [ ! -f .env ]; then \
		cp env.example .env; \
		echo "Created .env file from template"; \
	else \
		echo ".env file already exists"; \
	fi
	@echo "Installing dependencies..."
	go mod download
	@echo "Setup complete! Edit .env file with your configuration."

# Run database migrations
migrate:
	@echo "Running database migrations..."
	docker-compose up -d postgres redis
	@echo "Waiting for database to be ready..."
	@sleep 10
	@echo "Migrations completed!"

# Development mode
dev:
	docker-compose up -d postgres redis
	@sleep 5
	go run cmd/server/main.go

# Production mode
prod:
	docker-compose -f docker-compose.yml up -d

# Check application health
health:
	@echo "Checking application health..."
	@curl -f http://localhost:8080/health || echo "Application is not running"

# Generate API documentation
docs:
	@echo "Generating API documentation..."
	@echo "API documentation would be generated here"

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Database operations
db-reset:
	docker-compose down -v
	docker-compose up -d postgres redis
	@sleep 10
	@echo "Database reset complete!"

# Backup database
db-backup:
	docker-compose exec postgres pg_dump -U esim_user esim_db > backup_$(shell date +%Y%m%d_%H%M%S).sql

# Restore database
db-restore:
	@echo "Usage: make db-restore FILE=backup_file.sql"
	@if [ -z "$(FILE)" ]; then \
		echo "Please specify backup file: make db-restore FILE=backup_file.sql"; \
		exit 1; \
	fi
	docker-compose exec -T postgres psql -U esim_user esim_db < $(FILE)

# Show application status
status:
	@echo "Application Status:"
	@docker-compose ps
	@echo ""
	@echo "Recent logs:"
	@docker-compose logs --tail=20

# Clean everything
clean-all: clean
	docker-compose down -v
	docker system prune -f

# Help for specific commands
help-build:
	@echo "Build the Go application"
	@echo "Usage: make build"

help-run:
	@echo "Run the application locally"
	@echo "Usage: make run"
	@echo "Note: Make sure database is running first"

help-docker:
	@echo "Docker commands:"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run with Docker Compose"
	@echo "  docker-stop     - Stop Docker services"
	@echo "  docker-logs     - View Docker logs" 