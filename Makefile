# Simple Makefile for a Go project

# docker compose (v2 and v1) reads this instead of ./docker-compose.yml.
export COMPOSE_FILE := compose/docker-compose.yml

# Build the application
all: build test ## Build and test the Go application

build: ## Build the Go binary
	@echo "Building..."
	
	
	@go build -o main cmd/api/main.go

# Run the application
run: ## Run the Go application
	@go run cmd/api/main.go
# Create DB container
docker-run: ## Start the local Docker Compose stack
	@if docker compose up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
	fi

# Shutdown DB container
docker-down: ## Stop the local Docker Compose stack
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Test the application
test: ## Run Go tests
	@echo "Testing..."
	@go test ./... -v
# Integrations Tests for the application
itest: ## Run database package tests
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean: ## Remove the Go binary
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch: ## Run Go with live reload (Air)
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch docker-run docker-down itest

# Migration runtime: PostgreSQL is isolated from other local databases.
postgres-up: ## Start local PostgreSQL and wait until ready
	docker compose up -d --wait postgres

postgres-check: ## Check local PostgreSQL readiness
	docker compose exec -T postgres pg_isready -U career_dev -d career_dev

# Local observability: Alloy (OTLP :4317) -> Prometheus (:9090) -> Grafana (:3000).
otel-up: ## Start the local observability stack
	docker compose up -d alloy prometheus grafana

# Same stack without the Alloy container; run a host Alloy build with alloy-local.
# Only the Alloy container is removed: --remove-orphans would also drop postgres.
LOCAL_ALLOY_COMPOSE := compose/docker-compose.local-alloy.yml
ALLOY ?= $(HOME)/Code/oss/alloy/build/alloy

otel-up-local: ## Start observability with Alloy running on the host
	docker compose rm -sf alloy
	docker compose -f $(LOCAL_ALLOY_COMPOSE) up -d prometheus grafana

alloy-local: ## Run Alloy on the host
	PROMETHEUS_REMOTE_WRITE_URL=http://localhost:9090/api/v1/write \
		$(ALLOY) run compose/config/alloy/config.alloy --storage.path=tmp/alloy-data

migrate: ## Apply database migrations
	go run ./cmd/api migrate

check-db: ## Verify database connection and schema
	go run ./cmd/api check-db

go-dev: migrate ## Apply migrations and start the Go development server
	go run ./cmd/api serve

rebuild-projections: ## Rebuild book and company summaries
	go run ./cmd/api rebuild-projections

test-postgres: ## Run Go race tests against local PostgreSQL
	TEST_DATABASE_URL='postgres://career_dev:career_dev_local@127.0.0.1:5433/career_dev?sslmode=disable' go test -race ./...

.PHONY: postgres-up postgres-check otel-up otel-up-local alloy-local migrate check-db go-dev rebuild-projections test-postgres

homelab-check: ## Run homelab source checks
	./scripts/deploy-homelab.sh check

homelab-build: ## Build the homelab image
	./scripts/deploy-homelab.sh build

homelab-deploy: ## Build and deploy to the homelab
	./scripts/deploy-homelab.sh deploy

homelab-rollback: ## Restore the previous homelab deployment
	./scripts/deploy-homelab.sh rollback

.PHONY: homelab-check homelab-build homelab-deploy homelab-rollback

# Run these steps in order, including when make is invoked with -j.
dev: ## Install dependencies, set up .env, start PostgreSQL, build, and serve
	npm ci
	npm run setup
	$(MAKE) postgres-up
	npm run build
	$(MAKE) go-dev

help: ## Show available commands
	@awk 'BEGIN { print "Usage: make <command>\n" } /^[a-zA-Z0-9_-]+:.*## / { split($$0, parts, "## "); sub(/:.*/, "", parts[1]); printf "  %-22s %s\n", parts[1], parts[2] }' $(MAKEFILE_LIST)

.PHONY: dev help
