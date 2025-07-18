# The Orchestrator - AI Novel Generation System
# Build and installation Makefile

# Build configuration
BINARY_NAME=orc
CMD_PATH=./cmd/orc
BUILD_DIR=./bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"
GO_BUILD=go build $(LDFLAGS)

# XDG paths
XDG_CONFIG_HOME ?= $(HOME)/.config
XDG_DATA_HOME ?= $(HOME)/.local/share
XDG_BIN_HOME ?= $(HOME)/.local/bin

# Installation paths
INSTALL_BIN_DIR=$(XDG_BIN_HOME)
INSTALL_CONFIG_DIR=$(XDG_CONFIG_HOME)/orchestrator
INSTALL_DATA_DIR=$(XDG_DATA_HOME)/orchestrator

.PHONY: all build test clean install uninstall deps lint help

all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "Built binaries in $(BUILD_DIR)/"

# Run tests
test:
	@echo "Running tests..."
	go test -race -cover ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -race -cover -v ./...

# Run benchmark tests
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running basic checks..."; \
		go vet ./...; \
		go fmt ./...; \
	fi

# Install the binary and configuration
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_BIN_DIR)..."
	@mkdir -p $(INSTALL_BIN_DIR)
	@mkdir -p $(INSTALL_CONFIG_DIR)
	@mkdir -p $(INSTALL_DATA_DIR)/prompts
	
	# Install binary
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/
	chmod +x $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	
	# Install configuration if it doesn't exist
	@if [ ! -f $(INSTALL_CONFIG_DIR)/config.yaml ]; then \
		echo "Installing default config..."; \
		cp config.yaml.example $(INSTALL_CONFIG_DIR)/config.yaml; \
	fi
	
	# Install example env file
	@if [ ! -f $(INSTALL_CONFIG_DIR)/.env ]; then \
		echo "Installing example environment file..."; \
		cp .env.example $(INSTALL_CONFIG_DIR)/.env; \
	fi
	
	# Install prompt templates
	cp prompts/*.txt $(INSTALL_DATA_DIR)/prompts/ 2>/dev/null || echo "No prompt templates found to install"
	
	# Create symlink in go/bin if it exists (per user preferences)
	@if [ -d $(HOME)/go/bin ]; then \
		echo "Creating symlink in ~/go/bin..."; \
		ln -sf $(INSTALL_BIN_DIR)/$(BINARY_NAME) $(HOME)/go/bin/$(BINARY_NAME); \
	fi
	
	@echo "Installation complete!"
	@echo "Binary: $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Config: $(INSTALL_CONFIG_DIR)/config.yaml"
	@echo "Data: $(INSTALL_DATA_DIR)/"
	@echo ""
	@echo "Make sure $(INSTALL_BIN_DIR) is in your PATH"
	@echo "Add to your shell rc file: export PATH=\"$(INSTALL_BIN_DIR):\$$PATH\""

# Uninstall the binary and configuration
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	rm -f $(HOME)/go/bin/$(BINARY_NAME)
	@echo "Uninstalled binary. Configuration files left in $(INSTALL_CONFIG_DIR)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	go clean

# Development server (if applicable)
dev: build
	@echo "Running in development mode..."
	$(BUILD_DIR)/$(BINARY_NAME) -verbose -config ./config.yaml.example

# Create a release
release: clean test lint build-all
	@echo "Creating release $(VERSION)..."
	@mkdir -p releases
	tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64
	tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-amd64
	tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64
	zip -j releases/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo "Release packages created in releases/"

# Show help
help:
	@echo "The Orchestrator Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build          Build the binary"
	@echo "  build-all      Build for multiple platforms"
	@echo "  test           Run tests with race detection"
	@echo "  test-verbose   Run tests with verbose output"
	@echo "  bench          Run benchmark tests"
	@echo "  deps           Download dependencies"
	@echo "  deps-update    Update dependencies"
	@echo "  lint           Run linters"
	@echo "  install        Install binary and config (XDG-compliant)"
	@echo "  uninstall      Uninstall binary"
	@echo "  clean          Clean build artifacts"
	@echo "  dev            Run in development mode"
	@echo "  release        Create release packages"
	@echo "  help           Show this help"
	@echo ""
	@echo "XDG Installation Paths:"
	@echo "  Binary:     $(INSTALL_BIN_DIR)"
	@echo "  Config:     $(INSTALL_CONFIG_DIR)"
	@echo "  Data:       $(INSTALL_DATA_DIR)"