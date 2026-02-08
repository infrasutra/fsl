# FSL — Flux Schema Language Toolkit

.PHONY: build test lint install clean

# Build the CLI binary
build:
	@echo "Building fsl CLI..."
	@go build -o bin/fsl ./cmd/fsl

# Run all tests
test:
	@echo "Testing..."
	@go test ./... -v -race

# Run linter
lint:
	@golangci-lint run ./...

# Install CLI to $GOPATH/bin
install:
	@echo "Installing fsl CLI..."
	@go install ./cmd/fsl

# Clean build artifacts
clean:
	@rm -rf bin/

# Run LSP server (for testing)
lsp: build
	@./bin/fsl lsp --stdio

# Build with version info (for releases)
release-build:
	@go build -ldflags "-X github.com/infrasutra/fsl/cmd/fsl/cmd.Version=$(VERSION) -X github.com/infrasutra/fsl/cmd/fsl/cmd.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/fsl ./cmd/fsl
