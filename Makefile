.PHONY: generate-csp build-frontend build clean
.DEFAULT_GOAL := build

# Build the entire project
build: build-proxy setup-frontend

build-proxy:
	go build

# Build the Svelte frontend
build-frontend:
	cd webserver/dashboard/frontend && pnpm install && pnpm run build
	$(MAKE) generate-csp

# Remove build artifacts
clean:
	rm -f reservoir.exe
	rm -rf webserver/dashboard/frontend/build

# Generate Content Security Policy (CSP) header with script hashes
generate-csp:
	go generate ./webserver/dashboard/csp