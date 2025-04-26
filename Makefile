# Makefile (Simple Version)

# Variables
BINARY_NAME=app
CMD_PATH=./cmd/api/main.go
LINT_TOOL=golangci-lint

# Default target executed when running `make`
.DEFAULT_GOAL := help

# Targets
.PHONY: help lint fix build run tidy clean

help: ## Display this help screen
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

lint: ## Run linters (requires golangci-lint in PATH)
	@echo "==> Linting..."
	$(LINT_TOOL) run ./...

fix: ## Auto-fix lint issues and format code (requires golangci-lint, gofmt, goimports in PATH)
	@echo "==> Fixing and Formatting..."
	$(LINT_TOOL) run --fix ./...
	go fmt ./...
	goimports -w .

build: tidy ## Build the Go application binary
	@echo "==> Building..."
	# -ldflags="-s -w" strips debug info for smaller binary
	go build -ldflags="-s -w" -o $(BINARY_NAME) $(CMD_PATH)
	@echo "==> Binary '$(BINARY_NAME)' created."

run: build ## Build and run the application locally
	@echo "==> Running..."
	./$(BINARY_NAME)

tidy: ## Tidy Go module files
	@echo "==> Tidying modules..."
	go mod tidy

clean: ## Remove the built binary
	@echo "==> Cleaning..."
	rm -f $(BINARY_NAME)