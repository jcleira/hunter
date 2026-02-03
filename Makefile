.PHONY: all build test clean install uninstall extension lint fmt help

# Binary name
BINARY=hunter

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build directory
BUILD_DIR=build
EXTENSION_DIR=extension

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Default target
all: build

## build: Build the hunter CLI binary
build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/hunter

## build-all: Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/hunter
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/hunter

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/hunter
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/hunter

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/hunter

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## lint: Run linters
lint:
	@echo "Running linters..."
	$(GOVET) ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f $(BINARY)

## install: Install hunter to /usr/local/bin
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed! Run 'hunter --help' to get started."

## uninstall: Remove hunter from /usr/local/bin
uninstall:
	@echo "Uninstalling $(BINARY)..."
	@sudo rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstalled."

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## extension: Package the Chrome extension
extension:
	@echo "Packaging Chrome extension..."
	@mkdir -p $(BUILD_DIR)
	@# Convert SVG icons to PNG (requires rsvg-convert or similar)
	@if command -v rsvg-convert > /dev/null; then \
		echo "Converting icons..."; \
		rsvg-convert -w 16 -h 16 $(EXTENSION_DIR)/icons/icon16.svg -o $(EXTENSION_DIR)/icons/icon16.png; \
		rsvg-convert -w 48 -h 48 $(EXTENSION_DIR)/icons/icon48.svg -o $(EXTENSION_DIR)/icons/icon48.png; \
		rsvg-convert -w 128 -h 128 $(EXTENSION_DIR)/icons/icon128.svg -o $(EXTENSION_DIR)/icons/icon128.png; \
	else \
		echo "rsvg-convert not found, skipping icon conversion"; \
		echo "Install librsvg: brew install librsvg"; \
	fi
	@# Create zip for Chrome Web Store
	cd $(EXTENSION_DIR) && zip -r ../$(BUILD_DIR)/hunter-extension.zip . -x "*.svg"
	@echo "Extension packaged: $(BUILD_DIR)/hunter-extension.zip"

## extension-dev: Load extension in development mode (instructions)
extension-dev:
	@echo "To load the extension in Chrome:"
	@echo "1. Open chrome://extensions/"
	@echo "2. Enable 'Developer mode'"
	@echo "3. Click 'Load unpacked'"
	@echo "4. Select the 'extension' directory"

## release: Create a release (builds all platforms + extension)
release: clean build-all extension
	@echo "Creating release..."
	@mkdir -p $(BUILD_DIR)/release
	@cp $(BUILD_DIR)/$(BINARY)-* $(BUILD_DIR)/release/
	@cp $(BUILD_DIR)/hunter-extension.zip $(BUILD_DIR)/release/
	@echo "Release artifacts in $(BUILD_DIR)/release/"

## version: Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

## help: Show this help message
help:
	@echo "Hunter - Git Blame for AI"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'
