VERSION ?= $(shell tr -d '[:space:]' < .VERSION)
GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK_VERSION ?= v1.5.0
BINARY ?= hercules
IMAGE ?= hercules:$(VERSION)
HERCULES_PORT ?= 9100
LDFLAGS := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*##"; print "Hercules targets:"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: quality build ## Run all local quality gates and build the binary.

.PHONY: run
run: ## Run the example configuration.
	CGO_ENABLED=1 $(GO) run -ldflags='$(LDFLAGS)' ./cmd/hercules run --config hercules.yml

.PHONY: debug
debug: ## Run with debug logging enabled.
	DEBUG=true CGO_ENABLED=1 $(GO) run -ldflags='$(LDFLAGS)' ./cmd/hercules run --config hercules.yml

.PHONY: check
check: build ## Run a full startup preflight against the example configuration.
	./$(BINARY) check --config hercules.yml

.PHONY: build
build: ## Build for the current operating system and architecture.
	CGO_ENABLED=1 $(GO) build -trimpath -ldflags='$(LDFLAGS)' -o $(BINARY) ./cmd/hercules

.PHONY: install
install: ## Install Hercules into GOBIN or GOPATH/bin.
	CGO_ENABLED=1 $(GO) install -trimpath -ldflags='$(LDFLAGS)' ./cmd/hercules

.PHONY: mod
mod: ## Tidy and verify Go modules.
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: fmt
fmt: ## Format Go files.
	$(GO) fmt ./...
	$(GOLANGCI_LINT) fmt

.PHONY: fmt-check
fmt-check: ## Fail if formatting differs from the repository standard.
	$(GOLANGCI_LINT) fmt --diff

.PHONY: lint
lint: ## Run the strict lint profile.
	$(GOLANGCI_LINT) run ./...

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: test
test: ## Run all tests with the race detector.
	$(GO) test -race -count=1 ./...

.PHONY: test-cover
test-cover: ## Write and summarize race-enabled test coverage.
	$(GO) test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

.PHONY: vuln
vuln: ## Check reachable Go vulnerabilities.
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

.PHONY: quality
quality: fmt-check lint vet test vuln ## Run every source quality and security gate.

.PHONY: docker-build
docker-build: ## Build the container for the current Docker platform.
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) -f build/Dockerfile .

.PHONY: docker-run
docker-run: ## Run the container with the repository example mounted read-only.
	docker run --rm -p $(HERCULES_PORT):$(HERCULES_PORT) \
		-v "$(CURDIR)/hercules.yml:/etc/hercules/hercules.yml:ro" \
		-v "$(CURDIR)/assets:/etc/hercules/assets:ro" \
		-v "$(CURDIR)/hercules-packages:/etc/hercules/hercules-packages:ro" \
		$(IMAGE)

.PHONY: clean
clean: ## Remove only known generated build and coverage artifacts.
	rm -f ./$(BINARY) ./coverage.out
	rm -rf ./dist
