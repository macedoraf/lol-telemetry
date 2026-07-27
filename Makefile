# Makefile for lol-telemetry

DIST_DIR := dist
CLI_NATIVE := $(DIST_DIR)/lol-cli
DAEMON_NATIVE := $(DIST_DIR)/lol-daemon
CLI_WINDOWS := $(DIST_DIR)/lol-cli.exe
DAEMON_WINDOWS := $(DIST_DIR)/lol-daemon.exe

.PHONY: help build build-windows clean

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the native CLI and daemon binaries for the current platform
	@mkdir -p $(DIST_DIR)
	go build -o $(CLI_NATIVE) ./cmd/lol-cli
	go build -o $(DAEMON_NATIVE) ./cmd/lol-daemon
	@echo "Built $(CLI_NATIVE) and $(DAEMON_NATIVE)"

build-windows: ## Cross-compile the CLI and daemon binaries for Windows (amd64)
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $(CLI_WINDOWS) ./cmd/lol-cli
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $(DAEMON_WINDOWS) ./cmd/lol-daemon
	@echo "Built $(CLI_WINDOWS) and $(DAEMON_WINDOWS)"

clean: ## Remove the dist/ directory
	@rm -rf $(DIST_DIR)
	@echo "Removed $(DIST_DIR)"
