.PHONY: build test clean

PLUGIN_NAME := example
OUTPUT_DIR := ../../bin/plugins

build:
	@echo "Building $(PLUGIN_NAME) plugin..."
	@mkdir -p $(OUTPUT_DIR)
	go build -buildmode=plugin -o $(OUTPUT_DIR)/$(PLUGIN_NAME).so .

test:
	go test -v ./...

clean:
	rm -f $(OUTPUT_DIR)/$(PLUGIN_NAME).so