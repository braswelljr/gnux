BINARY  := gnux
BIN_DIR := ./bin
MODULE  := github.com/braswelljr/gnux
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X $(MODULE)/internal/common.Version=$(VERSION)"

WEB_DIR := web

# Where `make install/links` puts the per-tool links: GOBIN, else GOPATH/bin.
GOBIN   := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

HASGOCILINT := $(shell which golangci-lint 2> /dev/null)
ifdef HASGOCILINT
    GOLINT = golangci-lint
else
    GOLINT = bin/golangci-lint
endif

.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z0-9_/.-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ── Build ───────────────────────────────────────────────────────────────────
.PHONY: build
build: ## Build the gnux binary for the host platform into bin/
	go build -trimpath $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) .

.PHONY: build/all
build/all: export CGO_ENABLED = 0
build/all: ## Cross-compile for linux/darwin/windows (amd64+arm64), statically
	GOOS=linux   GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build -trimpath $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build -trimpath $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(BIN_DIR)/$(BINARY)-windows-amd64.exe .

.PHONY: links
links: build ## Link every tool name (rmx, cpx, ...) to bin/gnux
	@for t in $$($(BIN_DIR)/$(BINARY) --list); do ln -sf $(BINARY) $(BIN_DIR)/$$t; done
	@echo "linked $$($(BIN_DIR)/$(BINARY) --list | wc -l | tr -d ' ') tools in $(BIN_DIR)"

.PHONY: install
install: ## Install gnux with version ldflags
	go install -trimpath $(LDFLAGS) .

.PHONY: install/links
install/links: install ## Install gnux and link every tool name next to it
	@for t in $$($(GOBIN)/$(BINARY) --list); do ln -sf $(BINARY) $(GOBIN)/$$t; done
	@echo "linked $$($(GOBIN)/$(BINARY) --list | wc -l | tr -d ' ') tools in $(GOBIN)"

.PHONY: run
run: ## Run gnux from source (pass args with ARGS=..., e.g. ARGS="lsx -la")
	go run . $(ARGS)

# ── Dependencies ──────────────────────────────────────────────────────────────
.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: download
download: ## Download module dependencies
	go mod download

# ── Quality ─────────────────────────────────────────────────────────────────
.PHONY: test
test: ## Run tests with the race detector
	go test -race ./...

.PHONY: cover
cover: ## Run tests and write coverage.out
	go test -race -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLINT) run

# Go through golangci-lint's formatters (gofmt -s and goimports, configured in
# .golangci.yml), the website through Prettier: the same passes the pre-commit
# hook applies to staged files, over everything.
.PHONY: format
format: ## Format everything: the Go source, then the website
	$(GOLINT) fmt
	cd $(WEB_DIR) && pnpm run format

.PHONY: fix
fix: format ## Same as format

.PHONY: hooks
hooks: ## Install the Git hooks in lefthook.yml: format and lint on commit, docs check on push
	lefthook install

.PHONY: ci
ci: vet lint test docs/check ## Run the full CI suite locally

# ── Documentation ─────────────────────────────────────────────────────────────
.PHONY: docs
docs: ## Regenerate the reference (documentation/, all formats) and the website content
	go run ./tools/gendocs documentation all
	go run ./tools/gendocs $(WEB_DIR)/content/docs mdx

.PHONY: docs/check
docs/check: docs ## Fail if the committed docs are out of date
	@if [ -n "$$(git status --porcelain documentation $(WEB_DIR)/content/docs)" ]; then \
		echo "generated docs are out of date - commit the regenerated files"; \
		git status --porcelain documentation $(WEB_DIR)/content/docs; \
		exit 1; \
	fi

# ── Website (Fumadocs + playground) ───────────────────────────────────────────
.PHONY: web/wasm
web/wasm: ## Build gnux for the browser (web/public/gnux.wasm + wasm_exec.js)
	GOOS=js GOARCH=wasm go build -trimpath $(LDFLAGS) -o $(WEB_DIR)/public/gnux.wasm ./tools/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(WEB_DIR)/public/wasm_exec.js

.PHONY: web/docs
web/docs: ## Regenerate the website's documentation content
	go run ./tools/gendocs $(WEB_DIR)/content/docs mdx

.PHONY: web/install
web/install: ## Install the website's dependencies
	cd $(WEB_DIR) && pnpm install --frozen-lockfile

.PHONY: web/dev
web/dev: web/wasm web/docs ## Run the website locally with live reload
	cd $(WEB_DIR) && pnpm dev

.PHONY: web/build
web/build: web/wasm web/docs ## Build the website for production
	cd $(WEB_DIR) && pnpm build

.PHONY: web/test
web/test: web/wasm ## Test the playground's filesystem against the real wasm
	cd $(WEB_DIR) && pnpm test

# ── Docker ────────────────────────────────────────────────────────────────────
.PHONY: docker/build
docker/build: ## Build the Docker image (tag: gnux:$(VERSION))
	docker build --build-arg VERSION=$(VERSION) -t $(BINARY):$(VERSION) -t $(BINARY):latest .

.PHONY: docker/run
docker/run: ## Run gnux in Docker against the current directory (ARGS=...)
	docker run --rm -v "$$(pwd):/work" $(BINARY):latest $(ARGS)

# ── Clean ─────────────────────────────────────────────────────────────────────
.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) dist coverage.out
