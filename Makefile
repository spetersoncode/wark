# Wark Makefile
# Local-first CLI task management for AI agent orchestration

BINARY_NAME := wark
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags - inject version info into cli package
CLI_PKG := github.com/spetersoncode/wark/internal/cli
LDFLAGS := -ldflags "-X $(CLI_PKG).Version=$(VERSION) -X $(CLI_PKG).GitCommit=$(GIT_COMMIT) -X $(CLI_PKG).BuildDate=$(BUILD_TIME)"

# Directories
BUILD_DIR := ./build
INSTALL_DIR := /usr/local/bin
UI_DIR := ./ui
UI_DIST := $(UI_DIR)/dist
UI_EMBED := ./internal/server/ui

# Go settings
GOFLAGS := -trimpath

.PHONY: all build build-all build-go build-ui copy-ui test install uninstall clean clean-ui fmt lint vet dev help

# Default target
all: build

## build: Build everything (UI + Go binary) - the default for production builds
build: build-ui copy-ui build-go
	@echo "Full build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Alias for build (backwards compatibility)
build-all: build

## build-go: Build only the Go binary (for quick iteration during development)
build-go:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wark
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-ui: Build the frontend (Vite + React)
build-ui:
	@echo "Building UI..."
	@cd $(UI_DIR) && pnpm install --frozen-lockfile && pnpm build
	@echo "UI built: $(UI_DIST)"

## copy-ui: Copy built UI files to embed location
copy-ui:
	@echo "Copying UI to embed location..."
	@rm -rf $(UI_EMBED)
	@mkdir -p $(UI_EMBED)
	@cp -r $(UI_DIST)/* $(UI_EMBED)/
	@echo "UI copied to: $(UI_EMBED)"

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

## install-user: Install wark to ~/go/bin (standard Go bin, no sudo required)
install-user: build
	@echo "Installing $(BINARY_NAME) to ~/go/bin..."
	@mkdir -p ~/go/bin
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/$(BINARY_NAME)
	@echo "Installed: ~/go/bin/$(BINARY_NAME)"
	@echo "Make sure ~/go/bin is in your PATH"

## uninstall: Remove wark from /usr/local/bin
uninstall:
	@echo "Removing $(BINARY_NAME) from $(INSTALL_DIR)..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(UI_EMBED)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## clean-ui: Remove UI build artifacts
clean-ui:
	@echo "Cleaning UI..."
	@rm -rf $(UI_DIST)
	@rm -rf $(UI_DIR)/node_modules
	@rm -rf $(UI_EMBED)
	@echo "UI clean complete"

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

## run: Quick build and run wark with arguments (Go only, for development)
run: build-go
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## version: Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"

## dev: Start development servers (UI dev + Go server)
dev:
	@echo "Starting development environment..."
	@echo "Run these in separate terminals:"
	@echo "  Terminal 1 (UI):   cd ui && pnpm dev"
	@echo "  Terminal 2 (Go):   make build && ./build/wark serve"
	@echo ""
	@echo "UI dev server: http://localhost:5173 (proxies /api to :18080)"
	@echo "Go server:     http://localhost:18080"

## dev-ui: Start UI development server
dev-ui:
	@cd $(UI_DIR) && pnpm dev

## help: Show this help message
help:
	@echo "Wark - Local-first CLI task management for AI agent orchestration"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
