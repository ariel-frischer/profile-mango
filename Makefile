
.PHONY: help deps d install i test t test-go test-installer test-v test-coverage lint l lint-go lint-shell format f clean c build b bin run r check-agent-sources go-install install-global uninstall u release patch minor major prep-release worktree worktree-clean

MODULE_PATH=gitlab.com/ariel-frischer/profile-mango
BUILD_VERSION?=$(shell git tag --sort=-v:refname 2>/dev/null | head -1)
ifeq ($(BUILD_VERSION),)
  BUILD_VERSION=dev
endif
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags="-X ${MODULE_PATH}/internal/version.Version=${BUILD_VERSION} \
                   -X ${MODULE_PATH}/internal/version.Commit=${COMMIT} \
                   -X ${MODULE_PATH}/internal/version.BuildDate=${BUILD_DATE} \
                   -s -w"
WORKTREE_SCRIPT ?= scripts/worktree-setup.sh
BASE ?= $(shell git branch --show-current 2>/dev/null || echo HEAD)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	go mod download

d: deps ## Alias for deps

install: ## Install mango to GOPATH/bin
	go install ${LDFLAGS} ./cmd/mango/

i: install ## Alias for install

go-install: install ## Compatibility alias for install

install-global: install ## Compatibility alias for install

test: test-go test-installer ## Run Go and installer tests

t: test ## Alias for test

test-go: ## Run Go tests
	go test ./...

test-installer: ## Run offline installer fixture tests
	sh tests/install_test.sh

test-v: ## Run tests (verbose)
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -race -coverprofile=coverage.out ./...

lint: lint-go lint-shell ## Run all linters

l: lint ## Alias for lint

lint-go: ## Run Go linters
	@if command -v mise >/dev/null 2>&1 && [ -f mise.toml ]; then \
		mise exec -- golangci-lint run; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet"; \
		go vet ./...; \
		fi

lint-shell: ## Check POSIX shell scripts
	@if command -v shellcheck >/dev/null 2>&1; then \
		shellcheck --shell=sh install.sh tests/install_test.sh; \
	else \
		sh -n install.sh && sh -n tests/install_test.sh; \
		echo "shellcheck not installed, ran sh -n"; \
	fi

format: ## Format code
	go fmt ./...

f: format ## Alias for format

clean: ## Clean build artifacts
	go clean
	rm -rf bin/ coverage.out

c: clean ## Alias for clean

build: ## Build binary with version info
	go build ${LDFLAGS} -o bin/mango ./cmd/mango/

b: build ## Alias for build

bin: build ## Alias for build

run: ## Run main package
	go run ${LDFLAGS} ./cmd/mango/

r: run ## Alias for run

check-agent-sources: ## Check documented agent sources without writing changes
	go run ./cmd/mango agents check --manifest "$(or $(MANIFEST),docs/dev/agents/sources.json)"

worktree: ## Create or reuse an isolated agent worktree (BRANCH required)
	@test -n "$(BRANCH)" || (echo "BRANCH is required: make worktree BRANCH=agent/name [BASE=$$(git branch --show-current)]" >&2; exit 1)
	@bash "$(WORKTREE_SCRIPT)" "$(BRANCH)" "$(BASE)"

worktree-clean: ## Remove registered worktrees beneath .worktrees (preserves reports)
	@git worktree list --porcelain | awk '/^worktree .*\/.worktrees\// { sub(/^worktree /, ""); print }' | while read wt; do \
		echo "removing $$wt"; \
		git worktree remove --force "$$wt"; \
	done

uninstall: ## Uninstall mango
	@./uninstall.sh

u: uninstall ## Alias for uninstall

##@ Release
prep-release: ## Full release flow (usage: make prep-release VERSION=v0.1.0)
ifndef VERSION
	$(error VERSION is required, e.g. make prep-release VERSION=v1.0.0)
endif
	@./scripts/release.sh $(VERSION)

release: prep-release ## Run release preflight, create tag, and push (usage: make release VERSION=v1.0.0)

patch: ## Bump patch version and release
	$(eval CURRENT=$(shell git tag --sort=-v:refname | head -1 | sed 's/^v//'))
	$(eval NEXT=v$(shell echo $(CURRENT) | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}'))
	@echo "Bumping $(CURRENT) -> $(NEXT)"
	$(MAKE) release VERSION=$(NEXT)

minor: ## Bump minor version and release
	$(eval CURRENT=$(shell git tag --sort=-v:refname | head -1 | sed 's/^v//'))
	$(eval NEXT=v$(shell echo $(CURRENT) | awk -F. '{printf "%d.%d.0", $$1, $$2+1}'))
	@echo "Bumping $(CURRENT) -> $(NEXT)"
	$(MAKE) release VERSION=$(NEXT)

major: ## Bump major version and release
	$(eval CURRENT=$(shell git tag --sort=-v:refname | head -1 | sed 's/^v//'))
	$(eval NEXT=v$(shell echo $(CURRENT) | awk -F. '{printf "%d.0.0", $$1+1}'))
	@echo "Bumping $(CURRENT) -> $(NEXT)"
	$(MAKE) release VERSION=$(NEXT)
