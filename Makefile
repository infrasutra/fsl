# FSL — Flux Schema Language Toolkit

.PHONY: build test lint install clean

# Build the CLI binary
build:
	@echo "Building fluxcms CLI..."
	@go build -o bin/fluxcms ./cmd/fluxcms

# Run all tests
test:
	@echo "Testing..."
	@go test ./... -v -race

# Run linter
lint:
	@golangci-lint run ./...

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
