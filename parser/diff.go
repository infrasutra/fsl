package parser

import (
	"encoding/json"
	"fmt"
)

// ChangeType represents the type of schema change
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeRemoved  ChangeType = "removed"
	ChangeTypeModified ChangeType = "modified"
)

// ChangeKind represents what kind of element changed
type ChangeKind string

const (
	ChangeKindField     ChangeKind = "field"
	ChangeKindDecorator ChangeKind = "decorator"
	ChangeKindEnum      ChangeKind = "enum"
	ChangeKindRelation  ChangeKind = "relation"
	ChangeKindType      ChangeKind = "type" // Type-level changes (singleton, collection)
	ChangeKindComponent ChangeKind = "component"
)

// SchemaChange represents a single change between schema versions
type SchemaChange struct {
	Type      ChangeType `json:"type"` // added, removed, modified
	Kind      ChangeKind `json:"kind"` // field, decorator, enum, relation
	Path      string     `json:"path"` // e.g., "fields.title", "fields.author.decorators.maxLength"
	FieldName string     `json:"fieldName,omitempty"`
	OldValue  any        `json:"oldValue,omitempty"`
	NewValue  any        `json:"newValue,omitempty"`
	Breaking  bool       `json:"breaking"` // True if change could break existing data
	Message   string     `json:"message"`
}

// SchemaDiff represents the complete diff between two schema versions
type SchemaDiff struct {
	FromVersion  int            `json:"fromVersion"`
	ToVersion    int            `json:"toVersion"`
	FromChecksum string         `json:"fromChecksum"`
	ToChecksum   string         `json:"toChecksum"`
	Changes      []SchemaChange `json:"changes"`
	HasBreaking  bool           `json:"hasBreaking"` // True if any breaking changes
}

// DiffSchemas compares two compiled schemas and returns the differences
func DiffSchemas(from, to *CompiledSchema) *SchemaDiff {
	diff := &SchemaDiff{
		FromVersion:  from.Version,
		ToVersion:    to.Version,
		FromChecksum: from.Checksum,
		ToChecksum:   to.Checksum,
		Changes:      []SchemaChange{},
	}

	if from.Checksum == to.Checksum {
		return diff
	}

	diff.compareTypeLevel(from, to)

	diff.compareFieldSets("", from.Fields, to.Fields)
	diff.compareComponents(from, to)

	diff.compareEnums(from, to)

	diff.compareRelations(from, to)

	for _, change := range diff.Changes {
		if change.Breaking {
			diff.HasBreaking = true
			break
		}
	}

	return diff
}

func (d *SchemaDiff) compareTypeLevel(from, to *CompiledSchema) {

	if from.Singleton != to.Singleton {
		d.Changes = append(d.Changes, SchemaChange{
			Type:     ChangeTypeModified,
			Kind:     ChangeKindType,
			Path:     "singleton",
			OldValue: from.Singleton,
			NewValue: to.Singleton,
			Breaking: from.Singleton && !to.Singleton,
			Message:  fmt.Sprintf("singleton changed from %v to %v", from.Singleton, to.Singleton),
		})
	}

	if from.Collection != to.Collection {
		d.Changes = append(d.Changes, SchemaChange{
			Type:     ChangeTypeModified,
			Kind:     ChangeKindType,
			Path:     "collection",
			OldValue: from.Collection,
			NewValue: to.Collection,
			Breaking: false,
			Message:  fmt.Sprintf("collection name changed from '%s' to '%s'", from.Collection, to.Collection),
		})
	}
}

func (d *SchemaDiff) compareComponents(from, to *CompiledSchema) {
	fromComponents := make(map[string]*CompiledComponent, len(from.Components))
	toComponents := make(map[string]*CompiledComponent, len(to.Components))
	for i := range from.Components {
		fromComponents[from.Components[i].Name] = &from.Components[i]
	}
	for i := range to.Components {
		toComponents[to.Components[i].Name] = &to.Components[i]
	}
	for name := range fromComponents {
		if _, exists := toComponents[name]; !exists {
			d.Changes = append(d.Changes, SchemaChange{
				Type:     ChangeTypeRemoved,
				Kind:     ChangeKindComponent,
				Path:     fmt.Sprintf("components.%s", name),
				Breaking: true,
				Message:  fmt.Sprintf("section type '%s' was removed", name),
			})
		}
	}
	for name, toComponent := range toComponents {
		fromComponent, exists := fromComponents[name]
		if !exists {
			d.Changes = append(d.Changes, SchemaChange{
				Type:    ChangeTypeAdded,
				Kind:    ChangeKindComponent,
				Path:    fmt.Sprintf("components.%s", name),
				Message: fmt.Sprintf("section type '%s' was added", name),
			})
			continue
		}
		d.compareFieldSets(name+".", fromComponent.Fields, toComponent.Fields)
	}
}

func (d *SchemaDiff) compareFieldSets(prefix string, fromList, toList []CompiledField) {
	fromFields := make(map[string]*CompiledField)
	toFields := make(map[string]*CompiledField)

	for i := range fromList {
		fromFields[prefix+fromList[i].Name] = &fromList[i]
	}
	for i := range toList {
		toFields[prefix+toList[i].Name] = &toList[i]
	}

	for name, fromField := range fromFields {
		if _, exists := toFields[name]; !exists {
			d.Changes = append(d.Changes, SchemaChange{
				Type:      ChangeTypeRemoved,
				Kind:      ChangeKindField,
				Path:      fmt.Sprintf("fields.%s", name),
				FieldName: name,
				OldValue:  fromField,
				Breaking:  true,
				Message:   fmt.Sprintf("field '%s' was removed", name),
			})
		}
	}

	for name, toField := range toFields {
		fromField, exists := fromFields[name]
		if !exists {

			breaking := toField.Required && !hasDefault(toField)
			d.Changes = append(d.Changes, SchemaChange{
				Type:      ChangeTypeAdded,
				Kind:      ChangeKindField,
				Path:      fmt.Sprintf("fields.%s", name),
				FieldName: name,
				NewValue:  toField,
				Breaking:  breaking,
				Message:   fmt.Sprintf("field '%s' was added", name),
			})
			continue
		}

		d.compareFieldDetails(name, fromField, toField)
	}
}

func (d *SchemaDiff) compareFieldDetails(name string, from, to *CompiledField) {

	if from.Type != to.Type {
		d.Changes = append(d.Changes, SchemaChange{
			Type:      ChangeTypeModified,
			Kind:      ChangeKindField,
			Path:      fmt.Sprintf("fields.%s.type", name),
			FieldName: name,
			OldValue:  from.Type,
			NewValue:  to.Type,
			Breaking:  true,
			Message:   fmt.Sprintf("field '%s' type changed from %s to %s", name, from.Type, to.Type),
		})
	}

	if from.Required != to.Required {
		breaking := !from.Required && to.Required
		d.Changes = append(d.Changes, SchemaChange{
			Type:      ChangeTypeModified,
			Kind:      ChangeKindField,
			Path:      fmt.Sprintf("fields.%s.required", name),
			FieldName: name,
			OldValue:  from.Required,
			NewValue:  to.Required,
			Breaking:  breaking,
			Message:   fmt.Sprintf("field '%s' required changed from %v to %v", name, from.Required, to.Required),
		})
	}

	if from.Array != to.Array {
		d.Changes = append(d.Changes, SchemaChange{
			Type:      ChangeTypeModified,
			Kind:      ChangeKindField,
			Path:      fmt.Sprintf("fields.%s.array", name),
			FieldName: name,
			OldValue:  from.Array,
			NewValue:  to.Array,
			Breaking:  true,
			Message:   fmt.Sprintf("field '%s' array changed from %v to %v", name, from.Array, to.Array),
		})
	}

	if from.IsRelation != to.IsRelation || from.RelationTo != to.RelationTo {
		d.Changes = append(d.Changes, SchemaChange{
			Type:      ChangeTypeModified,
			Kind:      ChangeKindRelation,
			Path:      fmt.Sprintf("fields.%s.relation", name),
			FieldName: name,
			OldValue:  map[string]any{"isRelation": from.IsRelation, "relationTo": from.RelationTo},
			NewValue:  map[string]any{"isRelation": to.IsRelation, "relationTo": to.RelationTo},
			Breaking:  true,
			Message:   fmt.Sprintf("field '%s' relation configuration changed", name),
		})
	}

	if !equalStringSlices(from.InlineEnum, to.InlineEnum) {

		breaking := hasRemovedValues(from.InlineEnum, to.InlineEnum)
		d.Changes = append(d.Changes, SchemaChange{
			Type:      ChangeTypeModified,
			Kind:      ChangeKindField,
			Path:      fmt.Sprintf("fields.%s.inlineEnum", name),
			FieldName: name,
			OldValue:  from.InlineEnum,
			NewValue:  to.InlineEnum,
			Breaking:  breaking,
			Message:   fmt.Sprintf("field '%s' enum values changed", name),
		})
	}

	d.compareDecorators(name, from.Decorators, to.Decorators)
}

func (d *SchemaDiff) compareDecorators(fieldName string, from, to map[string]any) {

	for decName, fromVal := range from {
		if _, exists := to[decName]; !exists {
			breaking := isBreakingDecoratorRemoval(decName)
			d.Changes = append(d.Changes, SchemaChange{
				Type:      ChangeTypeRemoved,
				Kind:      ChangeKindDecorator,
				Path:      fmt.Sprintf("fields.%s.decorators.%s", fieldName, decName),
				FieldName: fieldName,
				OldValue:  fromVal,
				Breaking:  breaking,
				Message:   fmt.Sprintf("decorator @%s removed from field '%s'", decName, fieldName),
			})
		}
	}

	for decName, toVal := range to {
		fromVal, exists := from[decName]
		if !exists {
			breaking := isBreakingDecoratorAddition(decName, toVal)
			d.Changes = append(d.Changes, SchemaChange{
				Type:      ChangeTypeAdded,
				Kind:      ChangeKindDecorator,
				Path:      fmt.Sprintf("fields.%s.decorators.%s", fieldName, decName),
				FieldName: fieldName,
				NewValue:  toVal,
				Breaking:  breaking,
				Message:   fmt.Sprintf("decorator @%s added to field '%s'", decName, fieldName),
			})
			continue
		}

		if !equalValues(fromVal, toVal) {
			breaking := isBreakingDecoratorChange(decName, fromVal, toVal)
			d.Changes = append(d.Changes, SchemaChange{
				Type:      ChangeTypeModified,
				Kind:      ChangeKindDecorator,
				Path:      fmt.Sprintf("fields.%s.decorators.%s", fieldName, decName),
				FieldName: fieldName,
				OldValue:  fromVal,
				NewValue:  toVal,
				Breaking:  breaking,
				Message:   fmt.Sprintf("decorator @%s value changed on field '%s'", decName, fieldName),
			})
		}
	}
}

func (d *SchemaDiff) compareEnums(from, to *CompiledSchema) {
	fromEnums := make(map[string]*CompiledEnum)
	toEnums := make(map[string]*CompiledEnum)

	for i := range from.Enums {
		fromEnums[from.Enums[i].Name] = &from.Enums[i]
	}
	for i := range to.Enums {
		toEnums[to.Enums[i].Name] = &to.Enums[i]
	}

	for name := range fromEnums {
		if _, exists := toEnums[name]; !exists {
			d.Changes = append(d.Changes, SchemaChange{
				Type:     ChangeTypeRemoved,
				Kind:     ChangeKindEnum,
				Path:     fmt.Sprintf("enums.%s", name),
				OldValue: fromEnums[name],
				Breaking: true,
				Message:  fmt.Sprintf("enum '%s' was removed", name),
			})
		}
	}

	for name, toEnum := range toEnums {
		fromEnum, exists := fromEnums[name]
		if !exists {
			d.Changes = append(d.Changes, SchemaChange{
				Type:     ChangeTypeAdded,
				Kind:     ChangeKindEnum,
				Path:     fmt.Sprintf("enums.%s", name),
				NewValue: toEnum,
				Breaking: false,
				Message:  fmt.Sprintf("enum '%s' was added", name),
			})
			continue
		}

		if !equalStringSlices(fromEnum.Values, toEnum.Values) {
			breaking := hasRemovedValues(fromEnum.Values, toEnum.Values)
			d.Changes = append(d.Changes, SchemaChange{
				Type:     ChangeTypeModified,
				Kind:     ChangeKindEnum,
				Path:     fmt.Sprintf("enums.%s.values", name),
				OldValue: fromEnum.Values,
				NewValue: toEnum.Values,
				Breaking: breaking,
				Message:  fmt.Sprintf("enum '%s' values changed", name),
			})
		}
	}
}

func (d *SchemaDiff) compareRelations(from, to *CompiledSchema) {
	fromRels := make(map[string]*CompiledRelation)
	toRels := make(map[string]*CompiledRelation)

	for i := range from.Relations {
		fromRels[from.Relations[i].FieldName] = &from.Relations[i]
	}
	for i := range to.Relations {
		toRels[to.Relations[i].FieldName] = &to.Relations[i]
	}

	for name := range fromRels {
		if _, exists := toRels[name]; !exists {
			d.Changes = append(d.Changes, SchemaChange{
				Type:      ChangeTypeRemoved,
				Kind:      ChangeKindRelation,
				Path:      fmt.Sprintf("relations.%s", name),
				FieldName: name,
				OldValue:  fromRels[name],
				Breaking:  true,
				Message:   fmt.Sprintf("relation '%s' was removed", name),
			})
		}
	}

	for name, toRel := range toRels {
		if _, exists := fromRels[name]; !exists {
			d.Changes = append(d.Changes, SchemaChange{
				Type:      ChangeTypeAdded,
				Kind:      ChangeKindRelation,
				Path:      fmt.Sprintf("relations.%s", name),
				FieldName: name,
				NewValue:  toRel,
				Breaking:  false,
				Message:   fmt.Sprintf("relation '%s' was added", name),
			})
		}
	}
}

func hasDefault(field *CompiledField) bool {
	_, ok := field.Decorators[DecDefault]
	return ok
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasRemovedValues(from, to []string) bool {
	toSet := make(map[string]bool)
	for _, v := range to {
		toSet[v] = true
	}
	for _, v := range from {
		if !toSet[v] {
			return true
		}
	}
	return false
}

func equalValues(a, b any) bool {

	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

func isBreakingDecoratorRemoval(decName string) bool {

	switch decName {
	case DecRequired:
		return false
	case DecUnique:
		return false
	default:
		return false
	}
}

func isBreakingDecoratorAddition(decName string, value any) bool {
	switch decName {
	case DecRequired:
		return true
	case DecMaxLength, DecMinLength:
		return true
	case DecMin, DecMax:
		return true
	case DecPattern:
		return true
	case DecMinItems, DecMaxItems:
		return true
	default:
		return false
	}
}

func isBreakingDecoratorChange(decName string, from, to any) bool {
	switch decName {
	case DecMaxLength:

		fromLen, ok1 := toInt64Value(from)
		toLen, ok2 := toInt64Value(to)
		return ok1 && ok2 && toLen < fromLen
	case DecMinLength:

		fromLen, ok1 := toInt64Value(from)
		toLen, ok2 := toInt64Value(to)
		return ok1 && ok2 && toLen > fromLen
	case DecMax:

		fromMax, ok1 := toFloat64Value(from)
		toMax, ok2 := toFloat64Value(to)
		return ok1 && ok2 && toMax < fromMax
	case DecMin:

		fromMin, ok1 := toFloat64Value(from)
		toMin, ok2 := toFloat64Value(to)
		return ok1 && ok2 && toMin > fromMin
	case DecMaxItems:

		fromMax, ok1 := toInt64Value(from)
		toMax, ok2 := toInt64Value(to)
		return ok1 && ok2 && toMax < fromMax
	case DecMinItems:

		fromMin, ok1 := toInt64Value(from)
		toMin, ok2 := toInt64Value(to)
		return ok1 && ok2 && toMin > fromMin
	default:
		return false
	}
}

func toInt64Value(v any) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case float64:
		return int64(val), true
	default:
		return 0, false
	}
}

func toFloat64Value(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int64:
		return float64(val), true
	case int:
		return float64(val), true
	default:
		return 0, false
	}
}

// ToJSON returns the diff as a JSON string
func (d *SchemaDiff) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// Summary returns a human-readable summary of the changes
func (d *SchemaDiff) Summary() string {
	if len(d.Changes) == 0 {
		return "No changes detected"
	}

	var added, removed, modified int
	for _, c := range d.Changes {
		switch c.Type {
		case ChangeTypeAdded:
			added++
		case ChangeTypeRemoved:
			removed++
		case ChangeTypeModified:
			modified++
		}
	}

	result := fmt.Sprintf("%d change(s): %d added, %d removed, %d modified",
		len(d.Changes), added, removed, modified)

	if d.HasBreaking {
		result += " (contains breaking changes)"
	}

	return result
}
