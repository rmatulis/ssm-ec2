.PHONY: build clean install test fmt lint help

VERSION ?= dev
BINARY_NAME = ec2-ssm-connector
BUILD_DIR = build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o $(BINARY_NAME)
	@echo "Build complete: ./$(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo "All builds complete in $(BUILD_DIR)/"

install: ## Install the binary to /usr/local/bin
	@echo "Installing $(BINARY_NAME)..."
	go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o $(BINARY_NAME)
	sudo mv $(BINARY_NAME) /usr/local/bin/
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

test: ## Run tests
	go test -v ./...

fmt: ## Format code
	go fmt ./...

lint: ## Run linter (requires golangci-lint)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

deps: ## Download dependencies
	go mod download
	go mod tidy

version: ## Show current version
	@if [ -f $(BINARY_NAME) ]; then \
		./$(BINARY_NAME) --version; \
	else \
		echo "Binary not built. Run 'make build' first."; \
	fi

run-ec2: ## Run in EC2 mode (requires AWS_PROFILE env var)
	go run main.go --mode ec2 --profile $(AWS_PROFILE)

run-rds: ## Run in RDS mode (requires AWS_PROFILE env var)
	go run main.go --mode rds --profile $(AWS_PROFILE)
