# Flux Schema Language (FSL) Reference

Authoritative reference for the FSL parser/compiler and associated tooling.

## Quick Start

```fsl
@icon("newspaper")
@description("News articles")
type Article {
  title: String! @minLength(1) @maxLength(200) @searchable
  slug: String! @pattern("^[a-z0-9-]+$") @unique @index
  body: RichText
  slices: JSON! @slices(hero: HeroSlice, faq: FaqSlice, cta: CtaSlice)
  publishedAt: DateTime @index
  category: Category! @relation(inverse: "articles", onDelete: "restrict")
  tags: [String!]! @minItems(1) @maxItems(10)
  heroImage: Image @formats("jpg", "png", "webp") @maxSize(5000000)
  seo: JSON
}

type HeroSlice {
  heading: String!
  subheading: Text
  image: Image
}

type FaqSlice {
  title: String!
  items: JSON!
}

type CtaSlice {
  label: String!
  href: String!
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

### JSON

- `@slices(sliceType: ComponentType, ...)`

Rules:

- only valid on non-array `JSON` fields
- requires named mappings (`hero: HeroSlice`) instead of positional args
- mapped target must be another `type` in the same FSL document or in the section library (see below)
- runtime slice items are validated as `{ type: string, data: object }`
- `variation` is optional/free-form and not validated

### Field display

- `@label("Headline")`: the name editors see instead of the field name
- `@help("Keep it under 70 characters")`: a hint shown with the field
- `@placeholder("A short headline")`: example text shown in an empty input

Each takes one non-empty string. They are kept in the compiled field's decorators and do not affect validation.

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
  - Stored in compiled schema metadata. On a section type, it is stored on the compiled component, so editors can show what the section is for.

## Section Libraries

A section library is a definition whose types are shared sections. Any schema compiled with the library can list them in `@slices` without defining them:

```graphql
// sections.fsl
@description("Opening section with a heading and a button")
type Hero {
  heading: String! @label("Heading")
  button_link: String @pattern("^(/|#|https?://)")
}

@description("Questions and answers")
type Faq {
  questions: JSON! @slices(question: Question)
}

type Question {
  question: String!
  answer: Text!
}
```

```graphql
// landing.fsl
type Landing {
  title: String!
  sections: JSON! @slices(hero: Hero, faq: Faq)
}
```

- Library types are valid only as `@slices` targets, not as relation targets.
- They are compiled into the schema's `components` with `shared: true`, so the compiled schema stays self-contained: validation and code generation need nothing else.
- Nested `@slices` inside a library type resolve within the library.
- Named enums used by library types are added to the compiled schema's enums.
- A type defined in the schema wins over a library type with the same name. With `RejectShadowing`, defining it is an error instead: `type 'Hero' is already defined in the section library`.
- Removing a section type, removing a section field, or adding a required section field is reported as a breaking change by `DiffSchemas`.

In Go:

```go
library, err := parser.ParseLibrary(sectionsSource)
components, err := parser.CompileLibrary(library)
compiled, err := parser.ParseAndCompileWithOptions(pageSource, "Landing", "landing", false, parser.Options{
    Library:         library,
    RejectShadowing: true,
})
result := parser.ParseWithDiagnosticsAndOptions(pageSource, parser.Options{Library: library})
```

The `fluxcms` CLI and the language server treat the types of the other `.fsl` files in the schemas directory as the library, so a page file can use sections defined in `sections.fsl`.

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

### Slice Zone Value (`@slices`)

Must be an array of slice objects. Each item must include:

- `type`: one of the allowed mapping keys from `@slices(...)`
- `data`: object validated against the mapped component type fields

Example:

```json
[
  {
    "type": "hero",
    "data": {
      "heading": "Welcome to Flux",
      "subheading": "Composable page content"
    }
  },
  {
    "type": "faq",
    "data": {
      "title": "Common questions",
      "items": []
    }
  }
]
```

## Current Limitations and Gotchas

- `Reference(...)` field type syntax is not part of backend FSL parser.
  - Use type-name relations instead (`author: Author`).
- `Enum(...)` field type syntax is not part of backend parser.
  - Use inline enum (`"a" | "b"`) or named `enum`.
- Named enum field values are parsed and compiled, but runtime document validation is strictest with inline enums today.
  - Use inline enums if you need hard value enforcement at validation time.
- Parser allows multiple `type` definitions in one FSL file, but CMS schema storage is currently single-model per schema record.
- Reserved field names:
  - `id`
  - `createdAt`
  - `updatedAt`

## Validation API

Validate FSL and get diagnostics (line/column/source):

- `POST /api/v1/projects/{project_id}/schemas/validate`
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
