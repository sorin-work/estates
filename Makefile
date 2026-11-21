# Common tasks. Run `make help` for the list.

TEST_DATABASE_URL ?= postgres://estates:estates@localhost:5434/estates?sslmode=disable

.PHONY: help up down db api web test test-api test-web lint lint-api lint-web fmt backup restore

help:
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

up: ## Build and start the whole stack in Docker
	docker compose up --build -d

down: ## Stop the stack (data is kept)
	docker compose down

backup: ## Take a database backup now, into ./backups
	docker compose run --rm backup once

restore: ## Restore a backup: make restore FILE=backups/estates-....dump
	@test -n "$(FILE)" || { echo "usage: make restore FILE=backups/estates-....dump"; exit 1; }
	docker compose stop api
	docker compose exec -T db pg_restore -U estates -d estates --clean --if-exists --no-owner < "$(FILE)"
	docker compose start api

db: ## Start only Postgres, for running the API and web app locally
	docker compose up -d db

api: ## Run the API against the Docker database
	cd api && set -a && . ./.env && set +a && go run ./cmd/api

web: ## Run the Vite dev server
	cd web && npm run dev

test: test-api test-web ## Run all tests

test-api: ## API tests, including the database tests
	cd api && TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test ./...

test-web: ## Web tests
	cd web && npm test

lint: lint-api lint-web ## Lint and type check everything

lint-api:
	cd api && go vet ./... && golangci-lint run

lint-web:
	cd web && npm run lint && npm run typecheck && npm run format:check

fmt: ## Format Go and web code
	cd api && gofmt -w . && goimports -local github.com/sorin-work/estates -w .
	cd web && npm run format
