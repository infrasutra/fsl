# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Section libraries: `parser.Options` with `ExternalTypes` and `Library`, `ParseAndCompileWithOptions`, `CompileWithOptions`, `ParseWithDiagnosticsAndOptions`, `ValidateSchemaWithOptions`, `NewValidatorWithOptions`, `ParseLibrary` and `CompileLibrary`. `@slices` may reference library types; they compile into `components` with `shared: true`. A schema may not define a type the library already defines.
- `description` on compiled components, from `@description` on section types.
- `@label`, `@help` and `@placeholder` on fields.
- `DiffSchemas` reports section type and section field changes, including breaking ones.
- The CLI and language server resolve `@slices` targets from the other files in the schemas directory.

### Removed

- `ParseAndCompileWithExternalTypes`, `ParseWithDiagnosticsAndExternalTypes`, `ValidateSchemaWithExternalTypes` and `NewValidatorWithExternalTypes`. Pass `parser.Options{ExternalTypes: …}` to the `…WithOptions` functions instead.

- Open-source governance and contribution templates
- Security policy and support guidance
