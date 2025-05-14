# Hercules Makefile
# Defines all common build, test, and development tasks.

# Build variables.
VERSION := $(shell cat .VERSION 2>/dev/null || echo "dev")
GO := go
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
HERCULES_DIR := ./cmd/hercules
BINARY_NAME := hercules
ENV ?= $(shell whoami)
TEST_PROFILE := coverage.out
GOLANGCI_LINT_VERSION ?= v2.1.5
# Extract port from hercules.yml, default to 9100 if not found.
HERCULES_PORT := $(shell grep -o 'port: [0-9]\+' hercules.yml 2>/dev/null | awk '{print $$2}' || echo 9100)

# Fix for macOS linker warnings - use a simpler approach.
ifeq ($(shell uname),Darwin)
  # Use a single warning suppression flag that works on macOS.
  export CGO_LDFLAGS := -Wl,-w
endif

# Output colors.
COLOR_RESET = \033[0m
COLOR_BLUE = \033[34m
COLOR_GREEN = \033[32m
COLOR_RED = \033[31m

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(COLOR_BLUE)Hercules Makefile Help$(COLOR_RESET)\n"} \
		/^##@/ { printf "\n$(COLOR_BLUE)%s$(COLOR_RESET)\n", substr($$0, 5) } \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: lint test build ## Run lint, tests, and build.

##@ Development

.PHONY: run
run: ## Run Hercules locally.
	@printf "$(COLOR_BLUE)Running Hercules in development mode$(COLOR_RESET)\n"
	CGO_ENABLED=1 ENV=$(ENV) $(GO) run -ldflags="-X 'main.version=$(VERSION)'" $(HERCULES_DIR)

.PHONY: debug
debug: ## Run Hercules with debugging enabled.
	@printf "$(COLOR_BLUE)Running Hercules in debug mode$(COLOR_RESET)\n"
	CGO_ENABLED=1 DEBUG=1 ENV=$(ENV) $(GO) run -ldflags="-X 'main.version=$(VERSION)'" $(HERCULES_DIR)

.PHONY: mod
mod: ## Tidy and verify Go modules.
	@printf "$(COLOR_BLUE)Tidying Go modules$(COLOR_RESET)\n"
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: fmt
fmt: ## Format Go code.
	@printf "$(COLOR_BLUE)Formatting Go code$(COLOR_RESET)\n"
	$(GO) fmt ./...

##@ Build

.PHONY: build
build: ## Build Hercules binary.
	@printf "$(COLOR_BLUE)Building Hercules binary$(COLOR_RESET)\n"
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -ldflags="-X main.version=$(VERSION)" -o $(BINARY_NAME) $(HERCULES_DIR)

# Specific targets for different platforms to help with cross-compilation.
.PHONY: build-darwin
build-darwin: ## Build for macOS (darwin).
	@printf "$(COLOR_BLUE)Building Hercules for macOS$(COLOR_RESET)\n"
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="-X main.version=$(VERSION)" -o $(BINARY_NAME)-darwin-arm64 $(HERCULES_DIR)

.PHONY: build-linux
build-linux: ## Build for Linux.
	@printf "$(COLOR_BLUE)Building Hercules for Linux$(COLOR_RESET)\n"
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-X main.version=$(VERSION)" -o $(BINARY_NAME)-linux-amd64 $(HERCULES_DIR)

.PHONY: install
install: build ## Install Hercules binary.
	@printf "$(COLOR_BLUE)Installing Hercules binary$(COLOR_RESET)\n"
	mv $(BINARY_NAME) $(GOPATH)/bin/

.PHONY: clean
clean: ## Clean build artifacts.
	@printf "$(COLOR_BLUE)Cleaning build artifacts$(COLOR_RESET)\n"
	rm -f $(BINARY_NAME)
	rm -f $(TEST_PROFILE)
	rm -rf bin/
	find . -type f -name "*.test" -delete
	find . -type f -name "*.out" -delete
	find . -type d -name "testdata" -exec rm -rf {} +; 2>/dev/null || true

##@ Quality

.PHONY: lint
lint: ## Run linters.
	@which golangci-lint > /dev/null 2>&1 || { \
		printf "$(COLOR_BLUE)Installing golangci-lint using go install$(COLOR_RESET)\n"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		printf "$(COLOR_GREEN)golangci-lint installed successfully$(COLOR_RESET)\n"; \
	}
	@printf "$(COLOR_BLUE)Running linters$(COLOR_RESET)\n"
	golangci-lint run --config ./.github/golangci.yml ./pkg/... ./cmd/...

.PHONY: vet
vet: ## Run Go vet.
	@printf "$(COLOR_BLUE)Running go vet$(COLOR_RESET)\n"
	$(GO) vet -v ./...

##@ Testing

.PHONY: test
test: ## Run tests.
	@printf "$(COLOR_BLUE)Running tests$(COLOR_RESET)\n"
	$(GO) test -race ./pkg/... ./cmd/...

.PHONY: test-cover
test-cover: ## Run tests with coverage.
	@printf "$(COLOR_BLUE)Running tests with coverage$(COLOR_RESET)\n"
	$(GO) test -v -race -cover ./pkg/... ./cmd/... -coverprofile=$(TEST_PROFILE)
	$(GO) tool cover -func=$(TEST_PROFILE)
	
.PHONY: test-cover-html
test-cover-html: test-cover ## Run tests with coverage and open HTML report.
	@printf "$(COLOR_BLUE)Opening coverage report in browser$(COLOR_RESET)\n"
	$(GO) tool cover -html=$(TEST_PROFILE)

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker image.
	@printf "$(COLOR_BLUE)Building Docker image with port $(HERCULES_PORT)$(COLOR_RESET)\n"
	docker build --build-arg PORT=$(HERCULES_PORT) -t hercules:$(VERSION) -f build/Dockerfile.linux.arm64 .

.PHONY: docker-run
docker-run: ## Run Hercules in Docker with mounted configuration, assets, and packages.
	@printf "$(COLOR_BLUE)Running Hercules in Docker$(COLOR_RESET)\n"
	docker run --rm -p $(HERCULES_PORT):$(HERCULES_PORT) \
		-v $(PWD)/hercules.yml:/app/config/hercules.yml \
		-v $(PWD)/assets:/app/assets \
		-v $(PWD)/hercules-packages:/app/hercules-packages \
		hercules:$(VERSION)

.PHONY: docker-debug
docker-debug: ## Run Hercules in Docker with debug mode and mounted volumes.
	@printf "$(COLOR_BLUE)Running Hercules in Docker with debug enabled$(COLOR_RESET)\n"
	docker run --rm -p $(HERCULES_PORT):$(HERCULES_PORT) -e DEBUG=1 \
		-v $(PWD)/hercules.yml:/app/config/hercules.yml \
		-v $(PWD)/assets:/app/assets \
		-v $(PWD)/hercules-packages:/app/hercules-packages \
		hercules:$(VERSION)

##@ CI Tasks

.PHONY: ci-test
ci-test: ## Run tests for CI.
	$(GO) test -v -race -coverprofile=$(TEST_PROFILE) ./pkg/... ./cmd/...

.PHONY: ci-build
ci-build: ## Build for CI.
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-X main.version=$(VERSION)" -o $(BINARY_NAME) $(HERCULES_DIR)
