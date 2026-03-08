package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// helpers to build test fixtures

func makeField(name, typ string, required, array bool, decorators map[string]any) CompiledField {
	if decorators == nil {
		decorators = map[string]any{}
	}
	return CompiledField{Name: name, Type: typ, Required: required, Array: array, Decorators: decorators}
}

func makeSchema(checksum string, fields []CompiledField, enums []CompiledEnum, relations []CompiledRelation) *CompiledSchema {
	if fields == nil {
		fields = []CompiledField{}
	}
	if enums == nil {
		enums = []CompiledEnum{}
	}
	if relations == nil {
		relations = []CompiledRelation{}
	}
	return &CompiledSchema{
		Checksum:  checksum,
		Fields:    fields,
		Enums:     enums,
		Relations: relations,
	}
}

// ── checksum shortcut ──────────────────────────────────────────────────────

func TestDiffSchemas_SameChecksum_NoChanges(t *testing.T) {
	from := makeSchema("abc123", nil, nil, nil)
	to := makeSchema("abc123", nil, nil, nil)
	diff := DiffSchemas(from, to)
	assert.Empty(t, diff.Changes)
	assert.False(t, diff.HasBreaking)
}

// ── field added ────────────────────────────────────────────────────────────

func TestDiffSchemas_FieldAdded_Optional_NonBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", true, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{
		makeField("title", "String", true, false, nil),
		makeField("subtitle", "String", false, false, nil),
	}, nil, nil)
	diff := DiffSchemas(from, to)
	require := assert.New(t)
	var added *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].FieldName == "subtitle" && diff.Changes[i].Type == ChangeTypeAdded {
			added = &diff.Changes[i]
		}
	}
	require.NotNil(added, "expected subtitle added change")
	require.False(added.Breaking)
	require.False(diff.HasBreaking)
}

func TestDiffSchemas_FieldAdded_RequiredNoDefault_Breaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", true, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{
		makeField("title", "String", true, false, nil),
		makeField("slug", "String", true, false, nil), // required, no @default
	}, nil, nil)
	diff := DiffSchemas(from, to)
	var added *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].FieldName == "slug" && diff.Changes[i].Type == ChangeTypeAdded {
			added = &diff.Changes[i]
		}
	}
	assert.NotNil(t, added)
	assert.True(t, added.Breaking)
	assert.True(t, diff.HasBreaking)
}

func TestDiffSchemas_FieldAdded_RequiredWithDefault_NonBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", true, false, nil)}, nil, nil)
	withDefault := makeField("status", "String", true, false, map[string]any{DecDefault: "draft"})
	to := makeSchema("b", []CompiledField{makeField("title", "String", true, false, nil), withDefault}, nil, nil)
	diff := DiffSchemas(from, to)
	var added *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].FieldName == "status" && diff.Changes[i].Type == ChangeTypeAdded {
			added = &diff.Changes[i]
		}
	}
	assert.NotNil(t, added)
	assert.False(t, added.Breaking)
}

// ── field removed ──────────────────────────────────────────────────────────

func TestDiffSchemas_FieldRemoved_AlwaysBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{
		makeField("title", "String", true, false, nil),
		makeField("body", "Text", false, false, nil),
	}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", true, false, nil)}, nil, nil)
	diff := DiffSchemas(from, to)
	var removed *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].FieldName == "body" && diff.Changes[i].Type == ChangeTypeRemoved {
			removed = &diff.Changes[i]
		}
	}
	assert.NotNil(t, removed)
	assert.True(t, removed.Breaking)
	assert.True(t, diff.HasBreaking)
}

// ── field type changed ─────────────────────────────────────────────────────

func TestDiffSchemas_FieldTypeChanged_AlwaysBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("count", "String", false, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("count", "Int", false, false, nil)}, nil, nil)
	diff := DiffSchemas(from, to)
	var typeChange *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].FieldName == "count" && diff.Changes[i].Path == "fields.count.type" {
			typeChange = &diff.Changes[i]
		}
	}
	assert.NotNil(t, typeChange)
	assert.True(t, typeChange.Breaking)
	assert.True(t, diff.HasBreaking)
}

// ── required changed ───────────────────────────────────────────────────────

func TestDiffSchemas_RequiredChanged_OptionalToRequired_Breaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", false, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", true, false, nil)}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "fields.title.required" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_RequiredChanged_RequiredToOptional_NonBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", true, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", false, false, nil)}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "fields.title.required" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

// ── array changed ──────────────────────────────────────────────────────────

func TestDiffSchemas_ArrayChanged_AlwaysBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("tags", "String", false, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("tags", "String", false, true, nil)}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "fields.tags.array" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

// ── decorators ─────────────────────────────────────────────────────────────

func TestDiffSchemas_DecoratorAdded_Required_Breaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("email", "String", false, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("email", "String", false, false, map[string]any{DecRequired: true})}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindDecorator && diff.Changes[i].Type == ChangeTypeAdded {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_DecoratorAdded_MaxLength_Breaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("name", "String", false, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("name", "String", false, false, map[string]any{DecMaxLength: int64(100)})}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindDecorator && diff.Changes[i].Type == ChangeTypeAdded {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_DecoratorAdded_Icon_NonBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", false, false, nil)}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", false, false, map[string]any{DecIcon: "star"})}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindDecorator && diff.Changes[i].Type == ChangeTypeAdded {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

func TestDiffSchemas_DecoratorRemoved_NonBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", false, false, map[string]any{DecUnique: true})}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", false, false, nil)}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindDecorator && diff.Changes[i].Type == ChangeTypeRemoved {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

func TestDiffSchemas_DecoratorModified_MaxLength_Decreased_Breaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", false, false, map[string]any{DecMaxLength: int64(200)})}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", false, false, map[string]any{DecMaxLength: int64(100)})}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindDecorator && diff.Changes[i].Type == ChangeTypeModified {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_DecoratorModified_MaxLength_Increased_NonBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", false, false, map[string]any{DecMaxLength: int64(100)})}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", false, false, map[string]any{DecMaxLength: int64(200)})}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindDecorator && diff.Changes[i].Type == ChangeTypeModified {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

func TestDiffSchemas_DecoratorModified_MinLength_Increased_Breaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", false, false, map[string]any{DecMinLength: int64(2)})}, nil, nil)
	to := makeSchema("b", []CompiledField{makeField("title", "String", false, false, map[string]any{DecMinLength: int64(10)})}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindDecorator && diff.Changes[i].Type == ChangeTypeModified {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

// ── inline enum ────────────────────────────────────────────────────────────

func TestDiffSchemas_InlineEnum_ValueRemoved_Breaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{{Name: "status", Type: "String", InlineEnum: []string{"draft", "published", "archived"}}}, nil, nil)
	to := makeSchema("b", []CompiledField{{Name: "status", Type: "String", InlineEnum: []string{"draft", "published"}}}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "fields.status.inlineEnum" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_InlineEnum_ValueAdded_NonBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{{Name: "status", Type: "String", InlineEnum: []string{"draft", "published"}}}, nil, nil)
	to := makeSchema("b", []CompiledField{{Name: "status", Type: "String", InlineEnum: []string{"draft", "published", "scheduled"}}}, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "fields.status.inlineEnum" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

// ── enum ───────────────────────────────────────────────────────────────────

func TestDiffSchemas_EnumAdded_NonBreaking(t *testing.T) {
	from := makeSchema("a", nil, nil, nil)
	to := makeSchema("b", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft", "published"}}}, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindEnum && diff.Changes[i].Type == ChangeTypeAdded {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

func TestDiffSchemas_EnumRemoved_Breaking(t *testing.T) {
	from := makeSchema("a", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft", "published"}}}, nil)
	to := makeSchema("b", nil, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindEnum && diff.Changes[i].Type == ChangeTypeRemoved {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_EnumValuesChanged_ValueRemoved_Breaking(t *testing.T) {
	from := makeSchema("a", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft", "published", "archived"}}}, nil)
	to := makeSchema("b", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft", "published"}}}, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindEnum && diff.Changes[i].Type == ChangeTypeModified {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_EnumValuesChanged_ValueAdded_NonBreaking(t *testing.T) {
	from := makeSchema("a", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft", "published"}}}, nil)
	to := makeSchema("b", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft", "published", "scheduled"}}}, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindEnum && diff.Changes[i].Type == ChangeTypeModified {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

// ── relations ──────────────────────────────────────────────────────────────

func TestDiffSchemas_RelationAdded_NonBreaking(t *testing.T) {
	from := makeSchema("a", nil, nil, nil)
	to := makeSchema("b", nil, nil, []CompiledRelation{{FieldName: "author", TargetType: "Author"}})
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindRelation && diff.Changes[i].Type == ChangeTypeAdded {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

func TestDiffSchemas_RelationRemoved_Breaking(t *testing.T) {
	from := makeSchema("a", nil, nil, []CompiledRelation{{FieldName: "author", TargetType: "Author"}})
	to := makeSchema("b", nil, nil, nil)
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Kind == ChangeKindRelation && diff.Changes[i].Type == ChangeTypeRemoved {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

// ── type-level: singleton / collection ────────────────────────────────────

func TestDiffSchemas_SingletonChanged_TrueToFalse_Breaking(t *testing.T) {
	from := &CompiledSchema{Checksum: "a", Singleton: true}
	to := &CompiledSchema{Checksum: "b", Singleton: false}
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "singleton" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.True(t, change.Breaking)
}

func TestDiffSchemas_SingletonChanged_FalseToTrue_NonBreaking(t *testing.T) {
	from := &CompiledSchema{Checksum: "a", Singleton: false}
	to := &CompiledSchema{Checksum: "b", Singleton: true}
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "singleton" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

func TestDiffSchemas_CollectionNameChanged_NonBreaking(t *testing.T) {
	from := &CompiledSchema{Checksum: "a", Collection: "posts"}
	to := &CompiledSchema{Checksum: "b", Collection: "articles"}
	diff := DiffSchemas(from, to)
	var change *SchemaChange
	for i := range diff.Changes {
		if diff.Changes[i].Path == "collection" {
			change = &diff.Changes[i]
		}
	}
	assert.NotNil(t, change)
	assert.False(t, change.Breaking)
}

// ── HasBreaking summary flag ───────────────────────────────────────────────

func TestDiffSchemas_HasBreaking_FalseWhenNoBreaking(t *testing.T) {
	from := makeSchema("a", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft"}}}, nil)
	to := makeSchema("b", nil, []CompiledEnum{{Name: "Status", Values: []string{"draft", "published"}}}, nil)
	diff := DiffSchemas(from, to)
	assert.False(t, diff.HasBreaking)
}

func TestDiffSchemas_HasBreaking_TrueWhenAnyBreaking(t *testing.T) {
	from := makeSchema("a", []CompiledField{makeField("title", "String", false, false, nil)}, nil, nil)
	to := makeSchema("b", nil, nil, nil) // field removed
	diff := DiffSchemas(from, to)
	assert.True(t, diff.HasBreaking)
}

// ── helper: equalStringSlices ─────────────────────────────────────────────

func TestEqualStringSlices(t *testing.T) {
	assert.True(t, equalStringSlices(nil, nil))
	assert.True(t, equalStringSlices([]string{}, []string{}))
	assert.True(t, equalStringSlices([]string{"a", "b"}, []string{"a", "b"}))
	assert.False(t, equalStringSlices([]string{"a"}, []string{"b"}))
	assert.False(t, equalStringSlices([]string{"a", "b"}, []string{"a"}))
	assert.False(t, equalStringSlices([]string{"a"}, []string{"a", "b"}))
}

// ── helper: hasRemovedValues ───────────────────────────────────────────────

func TestHasRemovedValues(t *testing.T) {
	assert.False(t, hasRemovedValues([]string{"a", "b"}, []string{"a", "b", "c"})) // only added
	assert.True(t, hasRemovedValues([]string{"a", "b", "c"}, []string{"a", "b"}))  // c removed
	assert.False(t, hasRemovedValues(nil, []string{"a"}))                          // nothing from → nothing removed
	assert.True(t, hasRemovedValues([]string{"a"}, nil))                           // a removed
}

// ── helper: hasDefault ────────────────────────────────────────────────────

func TestHasDefault(t *testing.T) {
	withDefault := makeField("status", "String", true, false, map[string]any{DecDefault: "draft"})
	withoutDefault := makeField("status", "String", true, false, nil)
	assert.True(t, hasDefault(&withDefault))
	assert.False(t, hasDefault(&withoutDefault))
}

// ── Summary / metadata ────────────────────────────────────────────────────

func TestDiffSchemas_VersionsAndChecksums(t *testing.T) {
	from := &CompiledSchema{Version: 1, Checksum: "aaa"}
	to := &CompiledSchema{Version: 2, Checksum: "bbb"}
	diff := DiffSchemas(from, to)
	assert.Equal(t, 1, diff.FromVersion)
	assert.Equal(t, 2, diff.ToVersion)
	assert.Equal(t, "aaa", diff.FromChecksum)
	assert.Equal(t, "bbb", diff.ToChecksum)
}
