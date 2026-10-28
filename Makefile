.PHONY: help up down db-up db-down api-up api-down web-up web-down smoke run web seed mocks fmt lint test check

# Minimum statement coverage of the backend, enforced by `make test`.
COVERAGE_MIN ?= 90

# Load .env (if present) so run, seed and test see DATABASE_URL and JWT_SECRET.
-include .env
export

help: ## Show available targets
	@grep -E '^[a-z-]+:.*## ' $(firstword $(MAKEFILE_LIST)) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

up: ## Start everything in containers: database, API, demo data, web (needs .env)
	docker compose up -d --build --wait

down: ## Stop and remove every container (data is kept; add -v to docker compose down to erase it)
	docker compose down

db-up: ## Start only the database container
	docker compose up -d --wait --no-deps postgres

db-down: ## Stop only the database container
	docker compose rm -sf postgres

api-up: ## Start only the API container (it needs the database: make db-up)
	docker compose up -d --build --wait --no-deps api

api-down: ## Stop only the API container
	docker compose rm -sf api

web-up: ## Start only the web container
	docker compose up -d --build --wait --no-deps web

web-down: ## Stop only the web container
	docker compose rm -sf web

smoke: ## Check the running containers end to end (after make up)
	./scripts/smoke.sh

run: ## Run the API on the host, not in a container (applies migrations on start)
	cd backend && go run ./cmd/api

web: ## Run the frontend dev server on the host (proxies /api to the API on :8080)
	cd frontend && npm run dev

seed: ## Load the demo CSV data from the host (safe to repeat)
	cd backend && go run ./cmd/seed

fmt: ## Format backend and frontend
	cd backend && gofmt -w .
	cd frontend && npm run format

lint: ## Lint, format-check and type-check both apps, the API spec and the generated types
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint && npm run format:check && npm run typecheck && npm run check:api
	npx --yes @redocly/cli@2.71.2 lint docs/api/openapi.yaml

mocks: ## Regenerate test mocks (mockery)
	cd backend && mockery

test: ## Run all tests, enforcing backend coverage (needs the database: make db-up)
	cd backend && go test ./... -race -coverpkg=./... -coverprofile=coverage.out
	cd backend && ./scripts/check-coverage.sh coverage.out $(COVERAGE_MIN)
	cd frontend && npm test

check: lint test ## Everything CI runs
