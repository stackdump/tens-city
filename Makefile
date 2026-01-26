# Makefile for tens-city project

# Variables
GO := GOWORK=off go
GOFLAGS := -v
BINARIES := webserver
BIN_DIR := bin
CMD_DIR := cmd
STORE_DIR := data

# Build flags
LDFLAGS := -w -s

# Phony targets
.PHONY: all build test clean fmt vet lint help install run-webserver $(BINARIES)

# Default target
all: build

# Help target
help: ## Display this help message
	@echo "Tens City - Makefile targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build all binaries
build: $(BINARIES) ## Build webserver binary

# Build webserver binary
webserver: ## Build webserver binary
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ ./$(CMD_DIR)/$@

# Install binaries to $GOPATH/bin
install: ## Install webserver to $GOPATH/bin
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./$(CMD_DIR)/webserver

# Run tests
test: ## Run all tests
	$(GO) test -v ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Format code
fmt: ## Format Go code
	$(GO) fmt ./...

# Run go vet
vet: ## Run go vet
	$(GO) vet ./...

# Run linter (requires golangci-lint)
lint: ## Run golangci-lint (requires golangci-lint installed)
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# Clean build artifacts
clean: ## Clean build artifacts
	rm -f $(BINARIES)
	rm -f coverage.out coverage.html
	rm -rf $(BIN_DIR)
	$(GO) clean

# Clean test data directory
clean-data: ## Clean data directory (WARNING: removes all stored objects)
	rm -rf $(STORE_DIR)

# Download dependencies
deps: ## Download Go dependencies
	$(GO) mod download

# Tidy dependencies
tidy: ## Tidy Go dependencies
	$(GO) mod tidy

# Run webserver (requires build first)
run-webserver: webserver ## Build and run webserver
	./webserver -addr :8080 -store $(STORE_DIR) -content content/posts

# Development target - format, vet, test, build
dev: fmt vet test build ## Format, vet, test, and build (development workflow)

# Check everything before commit
check: fmt vet test ## Format, vet, and test code

# Quick build without optimization (faster for development)
quick: ## Quick build without optimization flags
	$(GO) build -o webserver ./$(CMD_DIR)/webserver
