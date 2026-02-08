# Flux CMS CLI

Concise reference for schema migration and diff behavior.

## Commands

- `fluxcms validate ./schemas`
  - Validates schema definitions at the path (file or directory).
  - Use `--format json` for machine-readable output.
  - Fails with a non-zero exit code when validation errors are found.

- `fluxcms migrate preview --schema=./schemas`
  - Produces a structural overview of detected types and enums.
  - Use `--format json` for machine-readable output.
  - Intended for quick review; not a full diff.

- `fluxcms migrate check --schema=./schemas`
  - Flags required fields that do not have defaults.
  - Use `--format json` for machine-readable output.
  - Exits non-zero when issues are found.

- `fluxcms migrate diff --from=./schemas/v1 --to=./schemas/v2 [--type=Post]`
  - Shows a schema diff between two schema paths using the schema diff engine.
  - Use `--format json` for machine-readable output.
  - Output prints the type header, `diff.Summary()`, and per-change lines.

- `fluxcms migrate generate --schema=./schemas --name="add_author_field"`
  - Creates a JSON migration scaffold with an empty `changes` list.
  - Use this before applying migrations in environments that track them.

- `fluxcms generate typescript --target=content --workspace-api-id=<id>`
  - Generates the public content SDK for a workspace API.
  - CMS SDKs must be generated via `GET /api/v1/cms/workspaces/{workspace_id}/sdk?target=cms` because schema IDs are required.

## SDK usage

- Content SDK (public API):

```ts
import { FluxClient } from './sdk';

const client = new FluxClient({ baseURL: 'https://api.example.com' }, 'workspace-api-id');
const posts = await client.post.list({ page: 1, perPage: 20 });
const post = await client.post.getBySlug('hello-world');
```

- CMS SDK (authenticated API):

```ts
import { FluxClient } from './sdk';

const client = new FluxClient({ baseURL: 'https://api.example.com', apiKey: 'jwt-or-api-key' }, 'workspace-uuid');
const created = await client.post.create({ title: 'Hello' }, { slug: 'hello' });
const updated = await client.post.update(created.id, { title: 'Hello again' }, { message: 'rename' });
```

## Diff behavior

- CLI preview output is a high-level structural summary only.
- Richer schema diffs are recorded when schema updates happen via the API.
- Fetch diff history from:
  - `GET /api/v1/cms/workspaces/{workspace_id}/schemas/{schema_id}/migrations`

## Examples

- Validate a workspace schema:
  - `fluxcms validate ./schemas`
  - Output: `Validation passed` or `Validation failed` with error details.

- Preview a migration:
  - `fluxcms migrate preview --schema=./schemas`
  - Output: summary of detected types and enums.

- Check required fields before generating:
  - `fluxcms migrate check --schema=./schemas`
  - Output: list of required fields missing defaults; exits non-zero on issues.

- Diff two schema versions:
  - `fluxcms migrate diff --from=./schemas/v1 --to=./schemas/v2 --type=Post`
  - Output: `Schema diff for type Post` then `3 change(s): 1 added, 1 removed, 1 modified`, followed by grouped per-change lines like `- kind=field path=fields.title breaking=true message=field 'title' was removed`.

- Generate migration artifacts:
  - `fluxcms migrate generate --schema=./schemas --name="add_author_field"`
  - Output: creates a JSON migration file with an empty `changes` list (scaffold).
