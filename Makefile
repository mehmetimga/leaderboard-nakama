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

# Data seeding and live testing
seed: ## Seed leaderboard with random data (20 users)
	@./scripts/seed_data.sh

live-feed: ## Start live feed (continuous score submission every 2s)
	@INTERVAL=2 ./scripts/live_feed.sh

feed-go: ## Run Go-based data feeder (--seed for initial data, otherwise live feed)
	@go run ./scripts/feed_data.go $(ARGS)

seed-go: ## Seed 50 users using Go feeder
	@go run ./scripts/feed_data.go --seed --seed-count=50

# Kafka data feeding (recommended)
kafka-feed: ## Start Kafka-based live feed (recommended)
	@go run ./scripts/kafka/producer.go $(ARGS)

kafka-seed: ## Seed 50 users via Kafka
	@go run ./scripts/kafka/producer.go --seed --seed-count=50

# Demo mode - 10 users with visible rank changes
demo: ## 🎮 Demo mode: 10 users with visible rank changes (recommended for testing)
	@echo "🎮 Starting Demo Mode..."
	@echo "📺 Open http://localhost:3000 to watch!"
	@echo ""
	@go run ./scripts/kafka/demo/main.go

demo-fast: ## 🚀 Fast demo: rank changes every 1 second
	@go run ./scripts/kafka/demo/main.go --interval=1s

demo-slow: ## 🐢 Slow demo: rank changes every 5 seconds
	@go run ./scripts/kafka/demo/main.go --interval=5s

# Full stack commands
stack-up: ## Start full stack (Kafka + Nakama + PostgreSQL + API + Web)
	$(DOCKER_COMPOSE) up -d
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 10
	@echo "✅ Stack is up! Services:"
	@echo "   - Web UI: http://localhost:3000"
	@echo "   - API: http://localhost:8080"
	@echo "   - Nakama Console: http://localhost:7351 (admin/password)"
	@echo "   - Kafka: localhost:9092"

stack-down: ## Stop full stack
	$(DOCKER_COMPOSE) down

stack-logs: ## View all logs
	$(DOCKER_COMPOSE) logs -f

# Development with Kafka
dev-kafka: ## Start dependencies with Kafka and run API locally
	$(DOCKER_COMPOSE) up -d postgres nakama zookeeper kafka
	@echo "⏳ Waiting for Kafka to be ready..."
	@sleep 15
	@echo "✅ Dependencies ready! Starting API..."
	KAFKA_ENABLED=true KAFKA_BROKERS=localhost:9092 $(GO) run ./cmd/api

# Health check
health: ## Check health of all services
	@echo "🔍 Checking services..."
	@curl -s http://localhost:8080/health | jq . || echo "❌ API not responding"
	@echo ""
	@curl -s http://localhost:7350/healthcheck | jq . || echo "❌ Nakama not responding"

# Quick test
test-realtime: ## Test real-time updates (run in separate terminal after stack-up)
	@echo "🧪 Testing real-time updates via Kafka..."
	@echo "📺 Open http://localhost:3000 in browser"
	@echo "⏳ Sending 10 scores via Kafka in 3 seconds..."
	@sleep 3
	@go run ./scripts/kafka/producer.go --seed --seed-count=10
	@echo "✅ Check browser for real-time updates!"
