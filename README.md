# FSL — Flux Schema Language

[![Go Reference](https://pkg.go.dev/badge/github.com/infrasutra/fsl.svg)](https://pkg.go.dev/github.com/infrasutra/fsl)
[![CI](https://github.com/infrasutra/fsl/actions/workflows/ci.yml/badge.svg)](https://github.com/infrasutra/fsl/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/infrasutra/fsl/branch/main/graph/badge.svg)](https://codecov.io/gh/infrasutra/fsl)
[![Go Report Card](https://goreportcard.com/badge/github.com/infrasutra/fsl)](https://goreportcard.com/report/github.com/infrasutra/fsl)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

**FSL** is a schema-first domain language for defining content models. It powers [Flux CMS](https://github.com/infrasutra/fluxcms) but can be used independently for any schema-driven project.

## Features

- **Parser & Compiler** — Full lexer, parser, AST, and compiler for `.fsl` files
- **Validation** — Schema validation with rich diagnostics (errors, warnings, hints)
- **LSP Server** — Language Server Protocol for editor integration (VS Code, Neovim, etc.)
- **SDK Codegen** — Generate type-safe TypeScript SDKs from schemas
- **Diffing** — Schema diff for migration support
- **Starter Templates** — Built-in templates (blog, ecommerce, news, portfolio)

## Quick Start

### Install CLI

```bash
go install github.com/infrasutra/fsl/cmd/fsl@latest
```

Or download a pre-built binary from [Releases](https://github.com/infrasutra/fsl/releases).

### Write a Schema

```fsl
@icon("newspaper")
@description("Blog articles")
type Article {
  title: String! @minLength(1) @maxLength(200) @searchable
  slug: String! @pattern("^[a-z0-9-]+$") @unique @index
  body: RichText @blocks("paragraph", "heading", "list", "image")
  publishedAt: DateTime @index
  category: Category! @relation(inverse: "articles", onDelete: "restrict")
  tags: [String!]! @minItems(1) @maxItems(10)
}

type Category {
  name: String!
  slug: String! @unique
  articles: [Article!] @relation(inverse: "category")
}
```

### Validate

```bash
fsl validate schema.fsl
```

### Generate TypeScript SDK

```bash
fsl generate typescript --schema ./schemas --output ./sdk --target content
```

### Editor Integration

Start the LSP server (used by editor extensions):

```bash
fsl lsp --stdio
```

## Go Library Usage

Use FSL as a Go library in your own projects:

```go
import (
    "github.com/infrasutra/fsl/parser"
    "github.com/infrasutra/fsl/sdk/typescript"
    "github.com/infrasutra/fsl/sdk"
)

// Parse and validate
schema, err := parser.Parse(fslSource)

// Compile
compiled, err := parser.Compile(schema, "Article", "article", false)

// Get diagnostics (for IDE integration)
result := parser.ParseWithDiagnostics(fslSource)

// Generate TypeScript SDK
gen := &typescript.Generator{}
output, err := gen.Generate(compiledSchemas, sdk.GeneratorConfig{
    PackageName: "my-sdk",
})
```

## Editor Extensions

- **VS Code**: [vscode-fsl](https://github.com/infrasutra/vscode-fsl)
- **Neovim**: [nvim-fsl](https://github.com/infrasutra/nvim-fsl)

## Package Structure

| Package | Description |
|---------|-------------|
| `parser` | FSL lexer, parser, AST, compiler, validator, diff |
| `lsp` | Language Server Protocol implementation |
| `sdk` | SDK code generation interfaces |
| `sdk/typescript` | TypeScript SDK generator |
| `template` | Template file parser (YAML/JSON/FSL) |
| `templates` | Built-in starter templates |
| `cmd/fsl` | CLI binary |

## Documentation

- [FSL Language Reference](docs/fsl.md)
- [CLI Usage](docs/cli.md)

## Community and Project Policies

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Support](SUPPORT.md)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

Additional attribution details are listed in [NOTICE](NOTICE).
