# Wark Makefile
# Local-first CLI task management for AI agent orchestration

BINARY_NAME := wark
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Directories
BUILD_DIR := ./build
INSTALL_DIR := /usr/local/bin

# Go settings
GOFLAGS := -trimpath

.PHONY: all build test install uninstall clean fmt lint vet help

# Default target
all: build

## build: Build the wark binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wark
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

## test-short: Run tests without verbose output
test-short:
	@echo "Running tests..."
	go test ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## install: Install wark to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed: $(INSTALL_DIR)/$(BINARY_NAME)"

## install-user: Install wark to ~/bin (no sudo required)
install-user: build
	@echo "Installing $(BINARY_NAME) to ~/bin..."
	@mkdir -p ~/bin
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) ~/bin/$(BINARY_NAME)
	@echo "Installed: ~/bin/$(BINARY_NAME)"
	@echo "Make sure ~/bin is in your PATH"

## uninstall: Remove wark from /usr/local/bin
uninstall:
	@echo "Removing $(BINARY_NAME) from $(INSTALL_DIR)..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## fmt: Format Go source code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## lint: Run golangci-lint (requires golangci-lint installed)
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## deps: Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## run: Build and run wark with arguments
run: build
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## version: Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"

## help: Show this help message
help:
	@echo "Wark - Local-first CLI task management for AI agent orchestration"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
