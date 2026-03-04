# FSL CLI

Concise reference for schema validation, diffing, migration scaffolding, and SDK generation.

## Core Commands

- `fsl validate <path>`
  - Validates one file or all `.fsl` files under a directory.
  - Use `--format=json` for machine-readable output.
  - Exits non-zero on validation failures.

- `fsl migrate preview --schema=./schemas`
  - Shows a structural preview of detected schema types and enums.
  - Use `--format=json` for machine-readable output.

- `fsl migrate check --schema=./schemas`
  - Detects potentially breaking required fields with no default value.
  - Exits non-zero when breaking issues are found.

- `fsl migrate diff --from=./schemas/v1 --to=./schemas/v2 --type=Post`
  - Compares schema versions and prints added/removed/modified changes.
  - Use `--format=json` for structured output.

- `fsl migrate generate --schema=./schemas --name="add_author_field"`
  - Creates a migration scaffold JSON file in a `migrations` folder.

- `fsl generate typescript --schema=./schemas --output=./sdk --target=content`
  - Generates a type-safe TypeScript SDK from FSL schemas.
  - `--target` supports `content` today. `cms` is intentionally blocked in CLI.

## Template Commands

- `fsl template list [--path=./templates] [--category=content]`
- `fsl template validate <file>`
- `fsl template convert <input> <output>`
- `fsl template import <directory-or-file>`
- `fsl template create <file>`
- `fsl template export <slug> [output-file]`

## SDK Example

```ts
import { FluxClient } from './sdk';

const client = new FluxClient({ baseURL: 'https://api.example.com' }, 'workspace-api-id');
const posts = await client.post.list({ page: 1, perPage: 20 });
const post = await client.post.getBySlug('hello-world');
```

## Notes

- Config file lookup uses `.fsl.yaml` or `.fsl.yml` from current directory upward.
- Legacy `.fluxcms.yaml` and `.fluxcms.yml` are still recognized for compatibility.
- Default schema directory is `./schemas`.
- Default TypeScript output directory is `./sdk`.
