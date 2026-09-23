.DEFAULT_GOAL := help

GO ?= go
BIN_DIR := bin
BINARY := $(BIN_DIR)/wonderfeed

.PHONY: help build test vet tidy format serve serve-down provider-image provider-status provider-health provider-up provider-down backup-create backup-list

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

# Patched overlay on top of the pinned release image (Postgres cold-start fixes).
provider-image: ## Build wonderfeed-ytzero:src-patch from deploy/ytzero/src-overlay
	docker build -f deploy/ytzero/Dockerfile.src-patch -t wonderfeed-ytzero:src-patch .

serve: provider-image ## process-compose TUI for PostgreSQL + YT Zero
	YTZERO_IMAGE=wonderfeed-ytzero:src-patch ./scripts/pc-up.sh

serve-down: ## Stop provider stack without removing volumes
	./scripts/pc-down.sh

provider-up: build provider-image ## Start PostgreSQL + YT Zero detached
	YTZERO_IMAGE=wonderfeed-ytzero:src-patch $(BINARY) provider up

provider-down: build ## Stop compose stack without removing volumes
	$(BINARY) provider down

provider-status: build ## Show provider pin and compose state
	$(BINARY) provider status

provider-health: build ## GET local YT Zero /api/health
	$(BINARY) provider health

backup-create: build ## Encrypted local backup (pg_dump + portable state)
	$(BINARY) backup create --local-only

backup-list: build ## List local (and optional S3) backups
	$(BINARY) backup list
