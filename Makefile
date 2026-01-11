.DEFAULT_GOAL := build

# Build the entire project
build: build-frontend generate-csp build-proxy

# Run all tests
test: build-frontend generate-csp
	go test -v ./...

build-proxy:
	go build -ldflags "-X 'reservoir/version.Version=$(shell git describe --tags --always --dirty)'"

# Generate Content Security Policy (CSP) header with script hashes
generate-csp:
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

.PHONY: build build-proxy generate-csp build-frontend dev-frontend clean
