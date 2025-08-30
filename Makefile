.PHONY: generate-csp build-frontend build clean
.DEFAULT_GOAL := build

# Build the entire project
build: setup-frontend
	go build

# Build frontend and generate CSP
setup-frontend: build-frontend generate-csp

# Build the Svelte frontend
build-frontend:
	cd webserver/dashboard/frontend && pnpm install && pnpm run build

# Remove build artifacts
clean:
	rm -f reservoir.exe
	rm -rf webserver/dashboard/frontend/build

# Generate Content Security Policy (CSP) header with script hashes
generate-csp:
	go generate ./webserver/dashboard/csp