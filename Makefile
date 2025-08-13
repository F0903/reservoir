.PHONY: build-frontend build clean
.DEFAULT_GOAL := build

# Build the entire project
build: build-frontend
	go build -o reservoir.exe

# Build the Svelte frontend
build-frontend:
	cd webserver/dashboard/frontend && pnpm install && pnpm run build

# Remove build artifacts
clean:
	rm -f reservoir.exe
	rm -rf webserver/dashboard/frontend/build