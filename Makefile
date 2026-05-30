# go-flashduty — developer tasks. Mirrors the flashcatcloud org convention
# (see flashduty-mcp-server) for fmt / gci / lint.

# Tooling
BUILD_DIR := bin
GOLANGCI_LINT_VERSION := v2.11.4
GOLANGCI_LINT := $(BUILD_DIR)/golangci-lint
GCI_VERSION := v0.13.5
GCI := $(BUILD_DIR)/gci

# Go
GOCMD := go
GOFMT := gofmt
MODULE := $(shell $(GOCMD) list -m)

# Source roots to format (whole module; gci skips generated files itself).
FMT_DIRS := .

.PHONY: all
all: check

# ---------------------------------------------------------------------------
# Build / generate / test
# ---------------------------------------------------------------------------

.PHONY: build
build: ## Compile all packages
	$(GOCMD) build ./...

.PHONY: generate
generate: ## Regenerate the typed service layer from the OpenAPI spec
	$(GOCMD) generate ./...

.PHONY: test
test: ## Run unit tests with the race detector
	$(GOCMD) test -race ./...

.PHONY: e2e
e2e: ## Run live E2E tests (needs FLASHDUTY_E2E_APP_KEY [+ FLASHDUTY_E2E_BASE_URL])
	@if [ -z "$$FLASHDUTY_E2E_APP_KEY" ] && [ -z "$$FLASHDUTY_APP_KEY" ]; then \
		echo "Error: set FLASHDUTY_E2E_APP_KEY (and optionally FLASHDUTY_E2E_BASE_URL)"; \
		exit 1; \
	fi
	$(GOCMD) test -v -tags e2e ./e2e/... -timeout 10m

# ---------------------------------------------------------------------------
# Formatting / linting
# ---------------------------------------------------------------------------

.PHONY: fmt
fmt: $(GCI) ## Format source and sort imports (gofmt -s + gci)
	$(GOFMT) -s -w $(FMT_DIRS)
	$(GCI) write --skip-generated -s standard -s default -s "prefix($(MODULE))" $(FMT_DIRS)

.PHONY: gci
gci: $(GCI) ## Sort imports only
	$(GCI) write --skip-generated -s standard -s default -s "prefix($(MODULE))" $(FMT_DIRS)

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## Run golangci-lint with --fix
	$(GOLANGCI_LINT) run --fix

# ---------------------------------------------------------------------------
# Aggregate / tooling
# ---------------------------------------------------------------------------

.PHONY: check
check: fmt lint test build ## Pre-push: fmt, lint, test, build

.PHONY: tools
tools: $(GOLANGCI_LINT) $(GCI) ## Install pinned dev tools into ./bin

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(GOLANGCI_LINT): $(BUILD_DIR)
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BUILD_DIR) $(GOLANGCI_LINT_VERSION); \
	fi

$(GCI): $(BUILD_DIR)
	@if [ ! -f "$(GCI)" ]; then \
		echo "Installing gci $(GCI_VERSION)..."; \
		GOBIN=$(CURDIR)/$(BUILD_DIR) $(GOCMD) install github.com/daixiang0/gci@$(GCI_VERSION); \
	fi

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
