.DEFAULT_GOAL := build

# Build the entire project
build: build-proxy

# Run the Go backend and Svelte frontend in development mode
dev:
	$(MAKE) --no-print-directory -j2 dev-backend dev-frontend

# Run all tests
test: generate-csp test-proxy test-frontend

ci: check-frontend lint-frontend test-frontend generate-csp vet test-proxy build-proxy

vet:
	go vet ./...

test-proxy: export CGO_ENABLED := 1
test-proxy:
	go test -v -race -count=1 ./...

frontend-install:
	cd webserver/dashboard/frontend && pnpm install --frozen-lockfile

check-frontend: frontend-install
	cd webserver/dashboard/frontend && pnpm run check

lint-frontend: frontend-install
	cd webserver/dashboard/frontend && pnpm run lint

test-frontend: frontend-install
	cd webserver/dashboard/frontend && pnpm run test

# Run all benchmarks
bench:
	go test -v -bench . -run ^$$ ./...

# Run cache performance comparison
bench-cache:
	go test -v -bench BenchmarkCacheComparison -run ^$$ ./cache

# The proxy build depends on the generated CSP hashes
build-proxy: generate-csp
	go build -ldflags "-X 'reservoir/version.Version=$(shell git describe --tags --always --dirty)'"

# Generating CSP hashes depends on having a fresh frontend build
generate-csp: build-frontend
	go generate ./webserver/dashboard/csp

# Build the Svelte frontend
build-frontend: frontend-install
	cd webserver/dashboard/frontend && pnpm run build

# Run the Svelte frontend in development mode
dev-frontend:
	cd webserver/dashboard/frontend && pnpm run dev $(ARGS)

# Run the Go backend in development mode
dev-backend:
	go run .

# Remove build artifacts
clean:
	rm -f reservoir.exe
	rm -rf webserver/dashboard/frontend/build

.PHONY: build dev dev-backend build-proxy generate-csp build-frontend frontend-install check-frontend lint-frontend dev-frontend test ci vet test-proxy test-frontend bench bench-cache clean
