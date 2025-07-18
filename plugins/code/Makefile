.PHONY: build test install clean

PLUGIN_NAME := code
OUTPUT_DIR := ../../bin/plugins
PLUGIN_FILE := $(OUTPUT_DIR)/$(PLUGIN_NAME).so

# Build the plugin as a shared library
build:
	@echo "Building $(PLUGIN_NAME) plugin..."
	@mkdir -p $(OUTPUT_DIR)
	go build -buildmode=plugin -o $(PLUGIN_FILE) .
	@echo "Plugin built: $(PLUGIN_FILE)"

# Build as standalone binary for testing
build-binary:
	@echo "Building $(PLUGIN_NAME) binary..."
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(PLUGIN_NAME)-plugin ./cmd/
	@echo "Binary built: $(OUTPUT_DIR)/$(PLUGIN_NAME)-plugin"

# Run tests
test:
	@echo "Testing $(PLUGIN_NAME) plugin..."
	go test -v ./...

# Install plugin to user directory
install: build
	@echo "Installing $(PLUGIN_NAME) plugin..."
	@mkdir -p ~/.local/share/orchestrator/plugins
	@cp $(PLUGIN_FILE) ~/.local/share/orchestrator/plugins/
	@cp manifest.yaml ~/.local/share/orchestrator/plugins/$(PLUGIN_NAME).yaml
	@echo "Plugin installed to ~/.local/share/orchestrator/plugins/"

# Clean build artifacts
clean:
	@echo "Cleaning $(PLUGIN_NAME) plugin..."
	@rm -f $(PLUGIN_FILE)
	@rm -f $(OUTPUT_DIR)/$(PLUGIN_NAME)-plugin

# Development mode - watch and rebuild
dev:
	@echo "Watching $(PLUGIN_NAME) plugin for changes..."
	@which reflex >/dev/null || (echo "Please install reflex: go install github.com/cespare/reflex@latest" && exit 1)
	reflex -s -r '\.go$$' -- make build