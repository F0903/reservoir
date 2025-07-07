.PHONY: build-frontend install-frontend dev-frontend
.PHONY: build clean run

# Build the Svelte frontend
build-frontend:
	cd dashboard/frontend && pnpm install && pnpm run build

# Install frontend dependencies only
install-frontend:
	cd dashboard/frontend && pnpm install

# Watch mode for development (if you add it to Svelte)
dev-frontend:
	cd dashboard/frontend && pnpm run dev

# Build the entire project
build: build-frontend
	go build -o apt-cacher-go.exe

# Remove build artifacts
clean:
	rm -f apt-cacher-go.exe
	rm -rf frontend/build

# Development mode - build and run
dev: build-frontend
	go run .

# Run the built binary
run: build
	./apt-cacher-go.exe


