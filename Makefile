.PHONY: run debug help
VERSION:=$(shell cat .VERSION)
DUCKTHEUS_DIR="./cmd/ducktheus/"

build:
	CGO_ENABLED=1 go build -ldflags="-X main.VERSION=$(VERSION)" -o ducktheus $(DUCKTHEUS_DIR)

run: ## Run ducktheus locally
	CGO_ENABLED=1 go run -ldflags="-X 'main.VERSION=x.x.dev'" $(DUCKTHEUS_DIR)

debug: ## Run ducktheus locally with debug
	CGO_ENABLED=1 DEBUG=1 go run -ldflags="-X 'main.VERSION=x.x.dev'" $(DUCKTHEUS_DIR)

help: ## Display makefile help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
