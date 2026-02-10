.PHONY: web-dev web-build build test

# Run Vite dev server (run Go API separately on :8080)
web-dev:
	cd web && bun run dev

# Build frontend for production
web-build:
	cd web && bun run build

# Build full binary (frontend + Go)
build: web-build
	go build -o vault-pilot ./cmd/server

# Run all tests
test:
	go test ./...
