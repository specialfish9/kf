INSTALL_DIR = /usr/local/bin/kf
DEFAULT_CONFIG = kf.yaml
CONFIG_HOME = $(HOME)/.config
ENTRY_POINT=$(CURDIR)/cmd/kf/

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help

run: ## Run the project
	go run $(ENTRY_POINT)
.PHONY: run

install: ## Install deps
	go get ./...
.PHONY: install

update: ## Update deps
	go mod tidy
	go get -u ./...
.PHONY: update

clean: ## Clean project
	go clean
.PHONY: clean

install-bin: ## Install the binary
	@go build -o $(INSTALL_DIR) .
.PHONY: install-bin

config: ## Copy the default configuration file
	@cp $(DEFAULT_CONFIG) $(CONFIG_HOME)
.PHONY: config
