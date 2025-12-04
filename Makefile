.PHONY: help build run test clean docker-up docker-down lint

# Variables
APP_NAME := leaderboard-api
GO := go
DOCKER_COMPOSE := docker-compose

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	$(GO) build -o bin/$(APP_NAME) ./cmd/api

run: ## Run the application locally
	$(GO) run ./cmd/api

test: ## Run tests
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-up: ## Start all services with Docker Compose
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop all services
	$(DOCKER_COMPOSE) down

docker-logs: ## View logs from all services
	$(DOCKER_COMPOSE) logs -f

docker-build: ## Build Docker image
	$(DOCKER_COMPOSE) build api

deps-up: ## Start only dependencies (Nakama, PostgreSQL)
	$(DOCKER_COMPOSE) up -d postgres nakama

deps-down: ## Stop dependencies
	$(DOCKER_COMPOSE) stop postgres nakama

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	$(GO) fmt ./...

mod-tidy: ## Tidy go modules
	$(GO) mod tidy

# Database operations
db-migrate: ## Run database migrations
	@echo "Migrations are run automatically on startup"

# Development helpers
dev: deps-up run ## Start dependencies and run locally

# Load testing
load-test-write: ## Run load test for score submission
	@echo "Running load test for score submission..."
	hey -n 1000 -c 50 -m POST \
		-H "Content-Type: application/json" \
		-d '{"leaderboard_id":"global_scores","user_id":"load-test-user","score":1000}' \
		http://localhost:8080/api/v1/scores

load-test-read: ## Run load test for leaderboard retrieval
	@echo "Running load test for leaderboard retrieval..."
	hey -n 1000 -c 50 \
		http://localhost:8080/api/v1/leaderboards/global_scores?type=live&limit=100

