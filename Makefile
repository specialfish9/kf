INSTALL_DIR = /usr/local/bin/
KF_BIN=kf
KF_ENTRY_POINT=$(CURDIR)/cmd/kf/
KL_BIN=kl
KL_ENTRY_POINT=$(CURDIR)/cmd/kl/

DEFAULT_CONFIG = kf.yaml
CONFIG_HOME = $(HOME)/.config
DEBUG_FLAGS = -v

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help

runf: ## Run kf
	go run $(KF_ENTRY_POINT) $(DEBUG_FLAGS)
.PHONY: runf

runl: ## Run kl
	go run $(KL_ENTRY_POINT) $(DEBUG_FLAGS)
.PHONY: runl

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

install-all: installf installl ## Install the binaries
.PHONY: install-all

installf:
	echo "Installing KF..."
	@go build -o $(INSTALL_DIR)$(KF_BIN) $(KF_ENTRY_POINT)
.PHONY: installf

installl:
	echo "Installing KL..."
	@go build -o $(INSTALL_DIR)$(KL_BIN) $(KL_ENTRY_POINT)
.PHONY: installl


config: ## Copy the default configuration file
	@cp $(DEFAULT_CONFIG) $(CONFIG_HOME)
.PHONY: config
