.PHONY: all build clean test test-coverage install seal webserver keygen run-webserver help

# Build output directory
BUILD_DIR := .
BIN_DIR := bin

# Binary names
SEAL := seal
WEBSERVER := webserver
KEYGEN := keygen

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Default target
all: build

## build: Build all binaries
build: seal webserver keygen

## seal: Build the seal CLI tool
seal:
	@echo "Building seal..."
	$(GOBUILD) -o $(BUILD_DIR)/$(SEAL) ./cmd/seal

## webserver: Build the webserver
webserver:
	@echo "Building webserver..."
	$(GOBUILD) -o $(BUILD_DIR)/$(WEBSERVER) ./cmd/webserver

## keygen: Build the keygen CLI tool
keygen:
	@echo "Building keygen..."
	$(GOBUILD) -o $(BUILD_DIR)/$(KEYGEN) ./cmd/keygen

## install: Install all binaries to $GOPATH/bin
install:
	@echo "Installing binaries to $(GOPATH)/bin..."
	$(GOCMD) install ./cmd/seal
	$(GOCMD) install ./cmd/webserver
	$(GOCMD) install ./cmd/keygen

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -cover ./...
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## clean: Remove build artifacts and test data
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(BUILD_DIR)/$(SEAL)
	rm -f $(BUILD_DIR)/$(WEBSERVER)
	rm -f $(BUILD_DIR)/$(KEYGEN)
	rm -f coverage.out coverage.html
	rm -f *.test *.out
	rm -rf $(BUILD_DIR)/data/
	@echo "Clean complete"

## run-webserver: Build and run the webserver (requires SUPABASE_JWT_SECRET)
run-webserver: webserver
	@if [ -z "$$SUPABASE_JWT_SECRET" ]; then \
		echo "Error: SUPABASE_JWT_SECRET environment variable is not set"; \
		echo "Usage: SUPABASE_JWT_SECRET=your-secret make run-webserver"; \
		exit 1; \
	fi
	@echo "Starting webserver on :8080..."
	./$(WEBSERVER) -addr :8080 -store data -public public

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

## tidy: Tidy go.mod and go.sum
tidy:
	@echo "Tidying go modules..."
	$(GOMOD) tidy

## fmt: Format Go source code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## lint: Run formatters and linters
lint: fmt vet
	@echo "Linting complete"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
