# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `ValidateData` checks named enum values, in plain fields, lists and sections.
- `[T!]` lists are optional; only `[T!]!` makes the list itself required, in validation, the compiled `required` decorator and the linter.
- TypeScript SDK: the content client calls `/api/v1/content/{api_id}` with the delivery API's `page`, `limit`, `locale`, `sort`, `filter`, `fields` and `include` parameters, and `list` returns the paginated response; the CMS client calls `/api/v1/projects/...`.
- TypeScript SDK: relations are typed `string | { id }`, rich text follows the editor's `content`/`attrs`/`marks` shape, `ContentItem` carries `included` and an optional `published_at` (preview drafts have none), nested sections get their own types, input lists keep their array types, the generated client type-checks with `--strict`, and type names from display names with spaces are valid identifiers.

## [0.3.0] - 2026-09-30

### Added

- Section libraries: `parser.Options` with `ExternalTypes` and `Library`, `ParseAndCompileWithOptions`, `CompileWithOptions`, `ParseWithDiagnosticsAndOptions`, `ValidateSchemaWithOptions`, `NewValidatorWithOptions`, `ParseLibrary` and `CompileLibrary`. `@slices` may reference library types; they compile into `components` with `shared: true`. A schema may not define a type the library already defines.
- `description` on compiled components, from `@description` on section types.
- `@label`, `@help` and `@placeholder` on fields.
- `DiffSchemas` reports section type and section field changes, including breaking ones.
- The CLI and language server resolve `@slices` targets from the other files in the schemas directory.
- `LinterConfig.Workspace`: the `unused-types` rule counts `@slices` targets and uses from the other files in the schemas directory.
- Open-source governance and contribution templates
- Security policy and support guidance

### Changed

- `DiffSchemas` reports removing a `@slices` key, pointing a key at another type, or adding `@slices` to an existing field as breaking.
- The `unused-types` lint rule message now reads "is not used by any relation or @slices field".

### Fixed

- The TypeScript and Go SDKs declare an enum shared by several content types once instead of once per content type.

### Removed

- `ParseAndCompileWithExternalTypes`, `ParseWithDiagnosticsAndExternalTypes`, `ValidateSchemaWithExternalTypes` and `NewValidatorWithExternalTypes`. Pass `parser.Options{ExternalTypes: …}` to the `…WithOptions` functions instead.
