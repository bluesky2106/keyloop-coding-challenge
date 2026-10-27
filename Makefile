.PHONY: help up down fmt lint test check

help: ## Show available targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

up: ## Start Postgres
	docker compose up -d --wait

down: ## Stop Postgres
	docker compose down

fmt: ## Format backend and frontend
	cd backend && gofmt -w .
	cd frontend && npm run format

lint: ## Lint, format-check and type-check both apps and the API spec
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint && npm run format:check && npm run typecheck
	npx --yes @redocly/cli@2.71.2 lint docs/api/openapi.yaml

test: ## Run all tests
	cd backend && go test ./... -race
	cd frontend && npm test

check: lint test ## Everything CI runs
