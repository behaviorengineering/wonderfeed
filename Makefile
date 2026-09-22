.DEFAULT_GOAL := help

GO ?= go
BIN_DIR := bin
BINARY := $(BIN_DIR)/wonderfeed

.PHONY: help build test vet tidy format serve serve-down provider-status provider-health

help: ## List available make verbs
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-24s %s\n", $$1, $$2}'

build: ## Build wonderfeed into bin/
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) ./cmd/wonderfeed

test: ## Run unit tests
	$(GO) test ./...

vet: ## Run go vet
	$(GO) vet ./...

tidy: ## Run go mod tidy
	$(GO) mod tidy

format: ## Format Go sources
	$(GO) fmt ./...

serve: build ## Start YT Zero via host compose overlay
	$(BINARY) provider up

serve-down: build ## Stop YT Zero without removing volumes
	$(BINARY) provider down

provider-status: build ## Show provider pin and compose state
	$(BINARY) provider status

provider-health: build ## GET local YT Zero /api/health
	$(BINARY) provider health
