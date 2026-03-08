# FSL — Flux Schema Language Toolkit

.PHONY: build test lint install clean

# Build the CLI binary
build:
	@echo "Building fluxcms CLI..."
	@go build -o bin/fluxcms ./cmd/fluxcms

# Lint the application
lint:
	@echo "Linting..."
	@go vet ./...
	@go tool staticcheck ./...

# format the application
fmt:
	@echo "formating..."
	@go tool gofumpt -w -extra .
	@echo "format completed"

# Run all tests
test:
	@echo "Testing..."
	@go test ./... -v -race

# Install CLI to $GOPATH/bin
install:
	@echo "Installing fluxcms CLI..."
	@go install ./cmd/fluxcms

# Clean build artifacts
clean:
	@rm -rf bin/

# Run LSP server (for testing)
lsp: build
	@./bin/fluxcms lsp --stdio

# Build with version info (for releases)
release-build:
	@go build -ldflags "-X github.com/infrasutra/fsl/cmd/fluxcms/cmd.Version=$(VERSION) -X github.com/infrasutra/fsl/cmd/fluxcms/cmd.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/fluxcms ./cmd/fluxcms
