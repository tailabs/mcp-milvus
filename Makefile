# MCP Milvus Makefile

.PHONY: help build test clean lint fmt docker run install deps dev tools release

# Variables
BINARY_NAME=mcp-milvus
REGISTRY?=ghcr.io/tailabs/mcp-milvus
PLATFORMS?=linux/amd64,linux/arm64
BUILD_DIR=build
DIST_DIR=dist
GO_VERSION=$(shell go version | cut -d ' ' -f 3)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
VERSION=$(shell git describe --tags --always --dirty)
RELEASE_VERSION=$(shell git describe --tags --always)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT)"

# Default target
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt: ## Format code
	@echo "Formatting code..."
	@gofmt -s -w .
	@go mod tidy

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build
build: ## Build binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mcp-milvus

install: build ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Cross-platform builds
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/mcp-milvus
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/mcp-milvus
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/mcp-milvus
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/mcp-milvus
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/mcp-milvus
	@echo "Built binaries in $(DIST_DIR)/"

# Docker
docker: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(REGISTRY):latest .
	@docker build -t $(REGISTRY):$(VERSION) .

docker-release-push:
	@echo "Building and push release docker image: $(REGISTRY):$(RELEASE_VERSION)"
	@docker buildx build --platform $(PLATFORMS) -t $(REGISTRY):$(RELEASE_VERSION) . --push

docker-run: docker ## Build and run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --name $(BINARY_NAME) --rm $(BINARY_NAME):latest

# Development server
run: build ## Build and run the application
	@echo "Starting $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run in development mode with live reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air for live reload..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Tools
tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/cosmtrek/air@latest

# Release
release: clean test build-all ## Prepare release (clean, test, build all platforms)
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && \
		tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
		tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
		tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
		tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
		zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release archives created in $(DIST_DIR)/"

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@go clean

# Info
info: ## Show build info
	@echo "Go version: $(GO_VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Version: $(VERSION)"
	@echo "Binary name: $(BINARY_NAME)" 