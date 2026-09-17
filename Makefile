# Simple Makefile for a Go project

# docker compose (v2 and v1) reads this instead of ./docker-compose.yml.
export COMPOSE_FILE := compose/docker-compose.yml

# Build the application
all: build test

build:
	@echo "Building..."
	
	
	@go build -o main cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go
# Create DB container
docker-run:
	@if docker compose up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
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
# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
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
postgres-up:
	docker compose up -d --wait postgres

postgres-check:
	docker compose exec -T postgres pg_isready -U career_dev -d career_dev

# Local observability: Alloy (OTLP :4317) -> Prometheus (:9090) -> Grafana (:3000).
otel-up:
	docker compose up -d alloy prometheus grafana

# Same stack without the Alloy container; run a host Alloy build with alloy-local.
# Only the Alloy container is removed: --remove-orphans would also drop postgres.
LOCAL_ALLOY_COMPOSE := compose/docker-compose.local-alloy.yml
ALLOY ?= $(HOME)/Code/oss/alloy/build/alloy

otel-up-local:
	docker compose rm -sf alloy
	docker compose -f $(LOCAL_ALLOY_COMPOSE) up -d prometheus grafana

alloy-local:
	PROMETHEUS_REMOTE_WRITE_URL=http://localhost:9090/api/v1/write \
		$(ALLOY) run compose/config/alloy/config.alloy --storage.path=tmp/alloy-data

migrate:
	go run ./cmd/api migrate

check-db:
	go run ./cmd/api check-db

go-dev: migrate
	go run ./cmd/api serve

seed-export:
	node scripts/export-seed.mjs

seed-dry-run:
	go run ./cmd/api seed --file .migration-private/seed-v2.json --source-store astro-seed --dry-run

seed-import:
	go run ./cmd/api seed --file .migration-private/seed-v2.json --source-store astro-seed

rebuild-projections:
	go run ./cmd/api rebuild-projections

test-postgres:
	TEST_DATABASE_URL='postgres://career_dev:career_dev_local@127.0.0.1:5433/career_dev?sslmode=disable' go test -race ./...

.PHONY: postgres-up postgres-check otel-up otel-up-local alloy-local migrate check-db go-dev seed-export seed-dry-run seed-import rebuild-projections test-postgres

homelab-check:
	./scripts/deploy-homelab.sh check

homelab-build:
	./scripts/deploy-homelab.sh build

homelab-deploy:
	./scripts/deploy-homelab.sh deploy

homelab-rollback:
	./scripts/deploy-homelab.sh rollback

.PHONY: homelab-check homelab-build homelab-deploy homelab-rollback
