SHELL := /bin/bash
BINARY := clickdown
WEB := web
GO ?= go
DIST := dist

# Resolve node/npm at make-parse time, preferring an nvm install if present.
NODE_BIN := $(firstword $(wildcard $(HOME)/.nvm/versions/node/*/bin) $(shell dirname $$(command -v node 2>/dev/null) 2>/dev/null))
ifneq ($(NODE_BIN),)
  export PATH := $(NODE_BIN):$(PATH)
endif
NPM ?= $(if $(NODE_BIN),$(NODE_BIN)/npm,npm)

.PHONY: help web web-install build build-mac build-mac-arm64 build-mac-amd64 build-windows build-all run dev dev-api dev-web clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-18s\033[0m %s\n",$$1,$$2}'

web-install: ## Install frontend dependencies
	$(NPM) --prefix $(WEB) install

web: ## Build the Vue SPA into web/dist (embedded by the Go binary)
	$(NPM) --prefix $(WEB) run build

build: web ## Build for the host platform
	mkdir -p $(DIST)
	$(GO) build -o $(DIST)/$(BINARY) .

build-mac: build-mac-arm64 build-mac-amd64 ## Build mac binaries for both arches

build-mac-arm64: web
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(DIST)/$(BINARY)-darwin-arm64 .

build-mac-amd64: web
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(DIST)/$(BINARY)-darwin-amd64 .

build-windows: web ## Build windows amd64 binary
	mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 $(GO) build -o $(DIST)/$(BINARY)-windows-amd64.exe .

build-all: build-mac build-windows ## Build all release binaries

run: build ## Build and run the production binary
	$(DIST)/$(BINARY)

dev: ## Run API + Vite dev server together (frontend proxies /api to :7878)
	@trap 'kill 0' INT TERM EXIT; \
	$(MAKE) -j2 dev-api dev-web

dev-api: ## Run the Go API only (no embedded frontend needed; serves placeholder)
	$(GO) run .

dev-web: ## Run the Vite dev server only (with /api proxy)
	$(NPM) --prefix $(WEB) run dev

clean:
	rm -rf $(DIST) $(WEB)/dist
