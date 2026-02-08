# Flux Schema Language (FSL) Reference

Authoritative reference for the FSL parser/compiler used by Flux CMS backend (`pkg/fsl`).

## Quick Start

```fsl
@icon("newspaper")
@description("News articles")
type Article {
  title: String! @minLength(1) @maxLength(200) @searchable
  slug: String! @pattern("^[a-z0-9-]+$") @unique @index
  body: RichText @blocks("paragraph", "heading", "list", "image")
  publishedAt: DateTime @index
  category: Category! @relation(inverse: "articles", onDelete: "restrict")
  tags: [String!]! @minItems(1) @maxItems(10)
  heroImage: Image @formats("jpg", "png", "webp") @maxSize(5000000)
  seo: JSON
}

type Category {
  name: String!
  slug: String! @unique
  articles: [Article!] @relation(inverse: "category")
}
```

## Grammar Overview

- Top-level definitions:
  - `type <Name> { ... }`
  - `enum <Name> { value1, value2, ... }`
- Type-level decorators can appear before `type`:
  - `@singleton`
  - `@collection("custom_collection_name")`
  - `@icon("lucide-icon-name")`
  - `@description("schema description")`
- Field syntax:
  - `<fieldName>: <Type>`
  - Required field: `<Type>!`
  - Array: `[<Type>]`
  - Required array elements: `[<Type>!]`
  - Required array itself: `[<Type>]!`

## Built-in Field Types

### Text Types

- `String`
  - Short single-line text.
  - Validation rejects newline characters.
- `Text`
  - Long plain text.
- `RichText`
  - Array of block objects at runtime.
  - Use `@blocks(...)` to limit allowed block types.

### Number Types

- `Int`
  - Whole number.
- `Float`
  - Decimal number.

### Date/Time Types

- `DateTime`
  - ISO 8601 / RFC3339 string (example: `2026-02-06T08:00:00Z`).
- `Date`
  - `YYYY-MM-DD` string.

### Other Types

- `Boolean`
- `JSON`
- `Image`
  - Asset object (must include `url`).
- `File`
  - Asset object (must include `url`).

## Enums

### Inline Enum

```fsl
type Article {
  status: "draft" | "published" | "archived"!
}
```

### Named Enum

```fsl
enum Status {
  draft,
  published,
  archived
}

type Article {
  status: Status!
}
```

## Relations

Relations are supported in two ways:

- Explicit relation decorator:
  - `author: Author! @relation`
- Auto-detected relation:
  - Any field whose type matches another content type name is treated as relation.
  - Example: `author: Author!` (without `@relation`) is still a relation.

### Relation Options

Use named args inside `@relation(...)`:

```fsl
author: Author! @relation(inverse: "articles", onDelete: "restrict")
```

- `inverse`: inverse field name in target type.
- `onDelete`: one of `cascade`, `restrict`, `setNull`.

## Field Decorators (Backend-Validated)

### String/Text

- `@maxLength(<int>)`
- `@minLength(<int>)`
- `@default(<value>)`
- `@searchable` (accepted on any field; mainly intended for text)

### String Only

- `@pattern("<regex>")`
- `@unique` (accepted on any field)
- `@index` (accepted on any field)

### Int/Float

- `@min(<number>)`
- `@max(<number>)`
- `@default(<number>)`
- `@index` (accepted on any field)

### Float Only

- `@precision(<int>)`

### Boolean

- `@default(true|false)`

### DateTime

- `@default(now)` (parser accepts identifier form)
- `@index` (accepted on any field)

### Date

- `@default(today)` (accepted)
- `@index` (accepted on any field)

### Arrays (any element type)

- `@minItems(<int>)`
- `@maxItems(<int>)`

### RichText

- `@blocks("paragraph", "heading", "list", ...)`

Valid block names:

- `paragraph`
- `heading`
- `blockquote`
- `code`
- `list`
- `image`
- `video`
- `embed`
- `table`
- `divider`
- `callout`
- `toggle`

### Image/File

- `@maxSize(<bytes>)`
- `@formats("jpg", "png", ...)`

For `Image`, formats must be in image set:

- `jpg`, `jpeg`, `png`, `gif`, `webp`, `svg`, `avif`

For `File`, formats can include image + document/archive/media formats:

- `jpg`, `jpeg`, `png`, `gif`, `webp`, `svg`, `avif`
- `pdf`, `doc`, `docx`, `xls`, `xlsx`, `ppt`, `pptx`
- `zip`, `tar`, `gz`
- `txt`, `csv`, `json`, `xml`
- `mp3`, `mp4`, `wav`, `avi`, `mov`

### General Flags

- `@hidden`

## Type-Level Decorators

- `@singleton`
  - Marks type as singleton.
- `@collection("name")`
  - Optional custom collection name.
- `@icon("lucide-name")`
  - Stored in compiled schema metadata.
- `@description("text")`
  - Stored in compiled schema metadata.

## Comments

Both comment styles are supported:

- `// line comment`
- `/* block comment */`

## Runtime Data Shapes

### RichText Value

Must be an array of objects containing at least `type`:

```json
[
  { "type": "paragraph", "children": [] },
  { "type": "heading", "level": 1 }
]
```

### Image/File Value

Must be an object with at least `url`:

```json
{
  "url": "https://cdn.example.com/file.jpg",
  "filename": "file.jpg",
  "size": 12345
}
```

### Relation Value

Accepted forms:

- UUID string
- object with UUID `id`

Examples:

```json
"550e8400-e29b-41d4-a716-446655440000"
```

```json
{ "id": "550e8400-e29b-41d4-a716-446655440000" }
```

## Current Limitations and Gotchas

- Decorator array literal syntax is **not** supported by backend parser.
  - Do this: `@blocks("paragraph", "heading")`
  - Not this: array-literal decorator arguments (for example `[...]`)
- `Reference(...)` field type syntax is not part of backend FSL parser.
  - Use type-name relations instead (`author: Author`).
- `Enum(...)` field type syntax is not part of backend parser.
  - Use inline enum (`"a" | "b"`) or named `enum`.
- Named enum field values are parsed and compiled, but runtime document validation is strictest with inline enums today.
  - Use inline enums if you need hard value enforcement at validation time.
- Field display decorators below are currently rejected by backend validator:
  - `@label`
  - `@help`
  - `@placeholder`
- Parser allows multiple `type` definitions in one FSL file, but CMS schema storage is currently single-model per schema record.
- Reserved field names:
  - `id`
  - `createdAt`
  - `updatedAt`

## Validation API

Validate FSL and get diagnostics (line/column/source):

- `POST /api/v1/cms/workspaces/{workspace_id}/schemas/validate`
- Body:

```json
{
  "definition": "type Article { title: String! }"
}
```

## Additional Examples

### Self-referential Hierarchy

```fsl
type Category {
  name: String!
  parent: Category
  children: [Category] @relation(inverse: "parent")
}
```

### Media-heavy Schema

```fsl
type Asset {
  title: String!
  image: Image @formats("jpg", "png", "webp") @maxSize(8000000)
  file: File @formats("pdf", "docx", "zip") @maxSize(20000000)
}
```
