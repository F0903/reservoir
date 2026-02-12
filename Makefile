.DEFAULT_GOAL := build

# Build the entire project
build: build-proxy

# Run all tests
test: generate-csp test-proxy test-frontend

test-proxy:
	CGO_ENABLED=1 go test -v -race -count=1 ./...

test-frontend:
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
build-frontend:
	cd webserver/dashboard/frontend && pnpm install && pnpm run build

# Run the Svelte frontend in development mode
dev-frontend:
	cd webserver/dashboard/frontend && pnpm run dev $(ARGS)

# Remove build artifacts
clean:
	rm -f reservoir.exe
	rm -rf webserver/dashboard/frontend/build

.PHONY: build build-proxy generate-csp build-frontend dev-frontend test test-proxy test-frontend bench bench-cache clean
