# Contributing to FSL

Thank you for your interest in contributing to FSL!

## Development Setup

```bash
git clone https://github.com/infrasutra/fsl.git
cd fsl
go mod download
make test
```

## Making Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests: `make test`
5. Commit with conventional prefixes: `feat:`, `fix:`, `docs:`, `chore:`
6. Open a Pull Request

## Pull Request Checklist

Before opening a PR, make sure you:

- Added or updated tests for behavior changes
- Updated docs when command behavior or public APIs changed
- Kept changes scoped to a single concern when possible
- Verified `go test ./...` passes locally

## Community Standards

- Read and follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- For security disclosures, follow [SECURITY.md](SECURITY.md) instead of filing a public issue
- For usage and troubleshooting questions, use [SUPPORT.md](SUPPORT.md)

## Code Style

- Go formatting: `gofmt` (tabs)
- Exported symbols: `CamelCase`
- Unexported symbols: `camelCase`
- Package names: `lowercase`

## Running Tests

```bash
make test       # All tests
go test ./parser/... -v   # Parser tests only
go test ./sdk/... -v      # SDK tests only
```

## Adding a New SDK Language

1. Create a new package under `sdk/` (e.g., `sdk/python/`)
2. Implement the `sdk.Generator` interface
3. Add a case in `cmd/fsl/cmd/generate.go`
4. Add tests

## Reporting Issues

Please include:
- FSL input that causes the issue
- Expected vs actual behavior
- Go version and OS
