BINARY_NAME ?= server
VERSION ?= $(if $(TAG),$(TAG),$(shell git describe --tags --always --dirty))
RELEASE_DIR ?= dist
RELEASE_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
DOCKER_IMAGE ?= ericlo417/connected-systems

.PHONY: help build run test test-coverage clean deps lint swag migrate docker-build docker-push docker-run docker-stop viewer-docker-build release-build validate-tag release

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

run: ## Run the application
	go run cmd/server/main.go

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf $(RELEASE_DIR)/
	rm -f coverage.out

deps: ## Download dependencies
	go mod download
	go mod tidy

lint: ## Run linter
	golangci-lint run

swag: ## Regenerate OpenAPI 3.0 spec from handler annotations
	swag init -g cmd/server/main.go -o docs/
	swagger2openapi docs/swagger.json -o docs/openapi.json

migrate: ## Run database migrations
	@echo "Database migrations not yet implemented"

docker-build: ## Build and push multi-arch Docker image (linux/amd64 + linux/arm64)
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag docker.monogatari.dev/connected-systems/cs-api-server \
		--push \
		.

docker-push: validate-tag ## Build and push Docker image to Docker Hub; usage: make docker-push TAG=v1.0.0
	docker build --tag $(DOCKER_IMAGE):$(TAG) .
	docker push $(DOCKER_IMAGE):$(TAG)

docker-run: ## Run Docker container
	docker-compose up

docker-stop: ## Stop Docker container
	docker-compose down

release-build: ## Build release binaries for multiple platforms
	rm -rf $(RELEASE_DIR)
	mkdir -p $(RELEASE_DIR)
	@for target in $(RELEASE_PLATFORMS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		output="$(RELEASE_DIR)/$(BINARY_NAME)-$(VERSION)-$$os-$$arch$$ext"; \
		echo "Building $$output"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags="-s -w" -o "$$output" cmd/server/main.go; \
	done
	cd $(RELEASE_DIR) && shasum -a 256 * > checksums.txt

validate-tag:
	@test -n "$(TAG)" || (echo "TAG is required. Usage: make release TAG=v1.0.0 or make docker-push TAG=v1.0.0" && exit 1)

release: validate-tag test release-build ## Create/push a git tag and GitHub release with binaries; usage: make release TAG=v1.0.0
	git tag -a $(TAG) -m "$(TAG)"
	git push origin $(TAG)
	gh release create $(TAG) $(RELEASE_DIR)/* --title "$(TAG)" --generate-notes

# ── cs-api-viewer ──────────────────────────────────────────────────────────────

viewer-docker-build: ## Build and push multi-arch Docker image for cs-api-viewer (linux/amd64 + linux/arm64)
	docker buildx build \
		--no-cache \
		--platform linux/amd64,linux/arm64 \
		--tag docker.monogatari.dev/connected-systems/cs-api-viewer \
		--file cs-api-viewer/Dockerfile \
		--push \
		.
