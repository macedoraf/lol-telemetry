# Makefile for lol-telemetry

DIST_DIR := dist
NATIVE_BIN := $(DIST_DIR)/lol-cli
WINDOWS_BIN := $(DIST_DIR)/lol-cli.exe

.PHONY: help build build-windows clean

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the native CLI binary for the current platform
	@mkdir -p $(DIST_DIR)
	go build -o $(NATIVE_BIN) ./cmd/lol-cli
	@echo "Built $(NATIVE_BIN)"

build-windows: ## Cross-compile the CLI binary for Windows (amd64)
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $(WINDOWS_BIN) ./cmd/lol-cli
	@echo "Built $(WINDOWS_BIN)"

clean: ## Remove the dist/ directory
	@rm -rf $(DIST_DIR)
	@echo "Removed $(DIST_DIR)"
