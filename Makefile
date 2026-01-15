# Makefile for AI Smart Queue project

COMPOSE = docker-compose
CONFIG_ENV ?= prod

# Select compose file based on environment
ifeq ($(CONFIG_ENV),prod)
    COMPOSE_FILE = docker-compose.prod.yml
else
    COMPOSE_FILE = docker-compose.dev.yml
endif

# Detect OS for script selection
ifeq ($(OS),Windows_NT)
    SCRIPT_EXT = .ps1
    SCRIPT_CMD = powershell -File
else
    SCRIPT_EXT = .sh
    SCRIPT_CMD = bash
endif

.PHONY: build up down logs ps restart test test-unit test-integration test-functional test-health test-quick clean

# Build all images
build:
	$(COMPOSE) -f $(COMPOSE_FILE) build

# Start all services
up:
	$(COMPOSE) -f $(COMPOSE_FILE) up -d

# Stop all services
down:
	$(COMPOSE) -f $(COMPOSE_FILE) down

# Show running containers
ps:
	$(COMPOSE) -f $(COMPOSE_FILE) ps

# Tail logs for all services
logs:
	$(COMPOSE) -f $(COMPOSE_FILE) logs -f

# Restart everything
restart: down up

# Clean everything (including volumes)
clean:
	$(COMPOSE) -f $(COMPOSE_FILE) down -v

# === Testing Commands ===

# Run all tests (unit + integration + functional)
test: test-unit test-integration test-health test-functional
	@echo "✅ All tests passed!"

# Run Go unit tests
test-unit:
	@echo "🧪 Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total

# Run integration tests (requires running services)
test-integration: up
	@echo "🔗 Running integration tests..."
	@sleep 5
	@go test -v -tags=integration ./...

# Run health checks on all services
test-health:
	@echo "🏥 Checking service health..."
	@$(SCRIPT_CMD) scripts/test-health$(SCRIPT_EXT)

# Run functional API tests
test-functional: up
	@echo "🎯 Running functional tests..."
	@sleep 5
	@$(SCRIPT_CMD) scripts/test-functional$(SCRIPT_EXT)

# Quick test (health + basic functionality)
test-quick: test-health
	@echo "⚡ Running quick functional test..."
	@$(SCRIPT_CMD) scripts/test-quick$(SCRIPT_EXT)

# Run tests with production config
test-prod:
	@echo "🏭 Testing with production config..."
	@CONFIG_ENV=prod $(MAKE) test-health test-quick
