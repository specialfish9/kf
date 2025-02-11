INSTALL_DIR = /usr/local/bin/kf
DEFAULT_CONFIG = kf.yaml
CONFIG_HOME = $(HOME)/.config

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.PHONY: help

run:
	go run .

install: update
.PHONY: install

update: ## Update the bin
	@go build -o $(INSTALL_DIR) .
.PHONY: update

config: ## Copy the default configuration file
	@cp $(DEFAULT_CONFIG) $(CONFIG_HOME)
.PHONY: config
