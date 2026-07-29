# Makefile for lol-telemetry

DIST_DIR := dist
CLI_NATIVE := $(DIST_DIR)/lol-cli
DAEMON_NATIVE := $(DIST_DIR)/lol-daemon
CLI_WINDOWS := $(DIST_DIR)/lol-cli.exe
DAEMON_WINDOWS := $(DIST_DIR)/lol-daemon.exe
ZIP_NAME := lol-telemetry-windows-amd64.zip

# -s -w: strip debug symbols and DWARF (smaller binary, less AV false positives)
# -trimpath: remove local filesystem paths from binary
LDFLAGS := -s -w
GOFLAGS := -trimpath

# Code signing (set CODE_SIGN_PFX and CODE_SIGN_PASS env vars)
SIGN_PFX := $(CODE_SIGN_PFX)
SIGN_PASS := $(CODE_SIGN_PASS)

.PHONY: help build build-windows build-windows-zip sign-windows clean

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
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(CLI_WINDOWS) ./cmd/lol-cli
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DAEMON_WINDOWS) ./cmd/lol-daemon
	@echo "Built $(CLI_WINDOWS) and $(DAEMON_WINDOWS)"

build-windows-zip: build-windows ## Build Windows artifacts and package into a single zip
	@cd $(DIST_DIR) && zip -9 $(ZIP_NAME) lol-cli.exe lol-daemon.exe
	@echo "Packaged $(DIST_DIR)/$(ZIP_NAME)"

sign-windows: build-windows ## Sign Windows .exe with osslsigncode (requires CODE_SIGN_PFX, CODE_SIGN_PASS)
	@command -v osslsigncode >/dev/null 2>&1 || { echo "osslsigncode not found. Install it first."; exit 1; }
	@echo "$(SIGN_PFX)" | base64 -d > $(DIST_DIR)/cert.pfx
	osslsigncode sign -pkcs12 $(DIST_DIR)/cert.pfx -pass "$(SIGN_PASS)" \
		-n "lol-telemetry" -i "https://github.com/rafael-macedo/lol-telemetry" \
		-in $(CLI_WINDOWS) -out $(CLI_WINDOWS).signed
	osslsigncode sign -pkcs12 $(DIST_DIR)/cert.pfx -pass "$(SIGN_PASS)" \
		-n "lol-telemetry" -i "https://github.com/rafael-macedo/lol-telemetry" \
		-in $(DAEMON_WINDOWS) -out $(DAEMON_WINDOWS).signed
	mv $(CLI_WINDOWS).signed $(CLI_WINDOWS)
	mv $(DAEMON_WINDOWS).signed $(DAEMON_WINDOWS)
	rm -f $(DIST_DIR)/cert.pfx
	@echo "Signed $(CLI_WINDOWS) and $(DAEMON_WINDOWS)"

clean: ## Remove the dist/ directory
	@rm -rf $(DIST_DIR)
	@echo "Removed $(DIST_DIR)"
