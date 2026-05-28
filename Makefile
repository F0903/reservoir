.DEFAULT_GOAL := build

GO ?= go
PNPM ?= pnpm

GO_PACKAGES := ./...
FRONTEND_DIR := webserver/dashboard/frontend
FRONTEND_BUILD_DIR := $(FRONTEND_DIR)/build
CSP_PACKAGE := ./webserver/dashboard/csp
AIR_CONFIG := .air.toml
VERSION ?= $(shell git describe --tags --always --dirty)
GO_BUILD_LDFLAGS = -X 'reservoir/version.Version=$(VERSION)'

# Build the entire project
.PHONY: build
build: build-proxy

# Run the Go backend and Svelte frontend in development mode
.PHONY: dev
dev: build-frontend
	$(MAKE) --no-print-directory -j2 dev-backend-watch dev-frontend

# Run all tests
.PHONY: test
test: generate-csp test-proxy test-frontend

.PHONY: ci
ci: check-frontend lint-frontend test-frontend generate-csp vet test-proxy build-proxy

.PHONY: vet
vet:
	$(GO) vet $(GO_PACKAGES)

.PHONY: test-proxy
test-proxy: export CGO_ENABLED := 1
test-proxy:
	$(GO) test -v -race -count=1 $(GO_PACKAGES)

.PHONY: frontend-install
frontend-install:
	cd $(FRONTEND_DIR) && $(PNPM) install --frozen-lockfile

.PHONY: check-frontend
check-frontend: frontend-install
	cd $(FRONTEND_DIR) && $(PNPM) run check

.PHONY: lint-frontend
lint-frontend: frontend-install
	cd $(FRONTEND_DIR) && $(PNPM) run lint

.PHONY: test-frontend
test-frontend: frontend-install
	cd $(FRONTEND_DIR) && $(PNPM) run test

# Run all benchmarks
.PHONY: bench
bench:
	$(GO) test -v -bench . -run ^$$ $(GO_PACKAGES)

# Run cache performance comparison
.PHONY: bench-cache
bench-cache:
	$(GO) test -v -bench BenchmarkCacheComparison -run ^$$ ./cache

# The proxy build depends on the generated CSP hashes
.PHONY: build-proxy
build-proxy: generate-csp
	$(GO) build -ldflags "$(GO_BUILD_LDFLAGS)"

# Generating CSP hashes depends on having a fresh frontend build
.PHONY: generate-csp
generate-csp: build-frontend
	$(GO) generate $(CSP_PACKAGE)

# Build the Svelte frontend
.PHONY: build-frontend
build-frontend: frontend-install
	cd $(FRONTEND_DIR) && $(PNPM) run build

# Run the Svelte frontend in development mode
.PHONY: dev-frontend
dev-frontend:
	cd $(FRONTEND_DIR) && $(PNPM) run dev $(ARGS)

# Run the Go backend in development mode with file watching
.PHONY: dev-backend-watch
dev-backend-watch:
	$(GO) tool air -c $(AIR_CONFIG)

# Run the Go backend once without file watching
.PHONY: dev-backend
dev-backend:
	$(GO) run .

# Remove build and runtime artifacts
.PHONY: clean
clean: clean-build clean-runtime

.PHONY: clean-build
clean-build:
	rm -f reservoir.exe
	rm -rf $(FRONTEND_BUILD_DIR)

.PHONY: clean-runtime
clean-runtime:
	rm -rf var/
