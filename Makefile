.PHONY: help build run test clean migrate viewer-docker-build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/server cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

deps: ## Download dependencies
	go mod download
	go mod tidy

lint: ## Run linter
	golangci-lint run

migrate: ## Run database migrations
	@echo "Database migrations not yet implemented"

docker-build: ## Build and push multi-arch Docker image (linux/amd64 + linux/arm64)
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag docker.monogatari.dev/connected-systems/cs-api-server \
		--push \
		.

docker-run: ## Run Docker container
	docker-compose up

docker-stop: ## Stop Docker container
	docker-compose down

# ── cs-api-viewer ──────────────────────────────────────────────────────────────

viewer-docker-build: ## Build and push multi-arch Docker image for cs-api-viewer (linux/amd64 + linux/arm64)
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag docker.monogatari.dev/connected-systems/cs-api-viewer \
		--file cs-api-viewer/Dockerfile \
		--push \
		.
