package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// CompiledSchema is the output format for storage
type CompiledSchema struct {
	Name        string              `json:"name"`
	SchemaID    string              `json:"schemaId,omitempty"`
	ApiID       string              `json:"apiId"`
	Singleton   bool                `json:"singleton"`
	Collection  string              `json:"collection,omitempty"`  // Custom collection name from @collection
	Icon        string              `json:"icon,omitempty"`        // Lucide icon name from @icon
	Description string              `json:"description,omitempty"` // Schema description from @description
	Fields      []CompiledField     `json:"fields"`
	Relations   []CompiledRelation  `json:"relations,omitempty"`  // Phase 2: relation metadata
	Enums       []CompiledEnum      `json:"enums,omitempty"`      // Phase 2: named enums used
	Components  []CompiledComponent `json:"components,omitempty"` // Typed component models used by @slices fields
	Version     int                 `json:"version"`
	Checksum    string              `json:"checksum"`
}

type CompiledField struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Required   bool           `json:"required"`
	Array      bool           `json:"array"`
	ArrayReq   bool           `json:"arrayRequired,omitempty"`
	Decorators map[string]any `json:"decorators,omitempty"`

	// Phase 2 fields
	InlineEnum []string            `json:"inlineEnum,omitempty"` // Inline enum values
	IsRelation bool                `json:"isRelation,omitempty"` // True if field is a relation
	RelationTo string              `json:"relationTo,omitempty"` // Target type for relations
	Slices     []CompiledSliceType `json:"slices,omitempty"`     // Typed slice-zone variants for JSON @slices
}

// CompiledSliceType maps runtime slice `type` to a component schema name.
type CompiledSliceType struct {
	Type   string `json:"type"`
	Schema string `json:"schema"`
}

// CompiledComponent captures fields for a reusable component type.
type CompiledComponent struct {
	Name   string          `json:"name"`
	Fields []CompiledField `json:"fields"`
}

// CompiledRelation holds relation metadata for the schema
type CompiledRelation struct {
	FieldName  string `json:"fieldName"`
	TargetType string `json:"targetType"`
	IsArray    bool   `json:"isArray"`
	IsRequired bool   `json:"isRequired"`
	Inverse    string `json:"inverse,omitempty"` // Inverse field name
	OnDelete   string `json:"onDelete,omitempty"`
}

// CompiledEnum holds enum definition for the schema
type CompiledEnum struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Compile converts parsed Schema to CompiledSchema
func Compile(schema *Schema, name, apiID string, singleton bool) (*CompiledSchema, error) {
	if len(schema.Types) == 0 {
		return nil, fmt.Errorf("schema must have at least one type definition")
	}

	// Find the type by name, or use the first type if name matches first type or is custom name
	var typeDef *TypeDef
	for i := range schema.Types {
		if schema.Types[i].Name == name {
			typeDef = &schema.Types[i]
			break
		}
	}
	// If no type with matching name found, use the first type
	// (for backward compatibility when name is a custom schema name)
	if typeDef == nil {
		typeDef = &schema.Types[0]
	}

	compiled := &CompiledSchema{
		Name:       name,
		ApiID:      apiID,
		Singleton:  singleton,
		Fields:     make([]CompiledField, 0, len(typeDef.Fields)),
		Relations:  []CompiledRelation{},
		Enums:      []CompiledEnum{},
		Components: []CompiledComponent{},
		Version:    1,
	}

	typeIndex := make(map[string]*TypeDef, len(schema.Types))
	for i := range schema.Types {
		typeIndex[schema.Types[i].Name] = &schema.Types[i]
	}

	// Check for type-level decorators
	for _, dec := range typeDef.Decorators {
		switch dec.Name {
		case DecCollection:
			if len(dec.Args) > 0 {
				if str, ok := dec.Args[0].(string); ok {
					compiled.Collection = str
				}
			}
		case DecSingleton:
			compiled.Singleton = true
		case DecIcon:
			if len(dec.Args) > 0 {
				if str, ok := dec.Args[0].(string); ok {
					compiled.Icon = str
				}
			}
		case DecDescription:
			if len(dec.Args) > 0 {
				if str, ok := dec.Args[0].(string); ok {
					compiled.Description = str
				}
			}
		}
	}

	// Compile named enums
	for _, enumDef := range schema.Enums {
		compiled.Enums = append(compiled.Enums, CompiledEnum{
			Name:   enumDef.Name,
			Values: enumDef.Values,
		})
	}

	// Compile fields
	componentTargets := make(map[string]bool)
	for _, field := range typeDef.Fields {
		compiledField, err := compileField(field)
		if err != nil {
			return nil, fmt.Errorf("failed to compile field '%s': %w", field.Name, err)
		}

		for _, sliceType := range compiledField.Slices {
			componentTargets[sliceType.Schema] = true
		}

		// Compile relation metadata
		if field.IsRelation {
			compiledField.RelationTo = field.Type

			relation := CompiledRelation{
				FieldName:  field.Name,
				TargetType: field.Type,
				IsArray:    field.Array,
				IsRequired: field.Required || field.ArrayReq,
			}

			// Extract relation options from decorator
			if args, ok := field.Decorators[DecRelation].(map[string]any); ok {
				if inverse, ok := args["inverse"].(string); ok {
					relation.Inverse = inverse
				}
				if onDelete, ok := args["onDelete"].(string); ok {
					relation.OnDelete = onDelete
				}
			}

			compiled.Relations = append(compiled.Relations, relation)
		}

		compiled.Fields = append(compiled.Fields, compiledField)
	}

	components, err := compileSliceComponents(componentTargets, typeIndex)
	if err != nil {
		return nil, err
	}
	compiled.Components = components

	// Compute checksum
	compiled.Checksum = ComputeChecksum(compiled)

	return compiled, nil
}

// ComputeChecksum generates SHA-256 checksum of the compiled schema
// Excludes version and checksum fields from the hash
func ComputeChecksum(cs *CompiledSchema) string {
	// Create a copy without version and checksum for hashing
	hashData := struct {
		Name        string              `json:"name"`
		ApiID       string              `json:"apiId"`
		Singleton   bool                `json:"singleton"`
		Collection  string              `json:"collection,omitempty"`
		Icon        string              `json:"icon,omitempty"`
		Description string              `json:"description,omitempty"`
		Fields      []CompiledField     `json:"fields"`
		Relations   []CompiledRelation  `json:"relations,omitempty"`
		Enums       []CompiledEnum      `json:"enums,omitempty"`
		Components  []CompiledComponent `json:"components,omitempty"`
	}{
		Name:        cs.Name,
		ApiID:       cs.ApiID,
		Singleton:   cs.Singleton,
		Collection:  cs.Collection,
		Icon:        cs.Icon,
		Description: cs.Description,
		Fields:      cs.Fields,
		Relations:   cs.Relations,
		Enums:       cs.Enums,
		Components:  cs.Components,
	}

	// Marshal to JSON for consistent hashing
	data, err := json.Marshal(hashData)
	if err != nil {
		// Fallback to empty string if marshaling fails
		return ""
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CompileMultiple compiles multiple type definitions into separate schemas
// This is for future use when we support multiple content models in one file
func CompileMultiple(schema *Schema, baseApiID string, singleton bool) ([]*CompiledSchema, error) {
	if len(schema.Types) == 0 {
		return nil, fmt.Errorf("schema must have at least one type definition")
	}

	compiled := make([]*CompiledSchema, 0, len(schema.Types))

	typeIndex := make(map[string]*TypeDef, len(schema.Types))
	for i := range schema.Types {
		typeIndex[schema.Types[i].Name] = &schema.Types[i]
	}

	for i, typeDef := range schema.Types {
		// Generate unique API ID for each type
		apiID := fmt.Sprintf("%s_%d", baseApiID, i)
		if baseApiID == "" {
			apiID = fmt.Sprintf("type_%d", i)
		}

		cs := &CompiledSchema{
			Name:       typeDef.Name,
			ApiID:      apiID,
			Singleton:  singleton,
			Fields:     make([]CompiledField, 0, len(typeDef.Fields)),
			Components: []CompiledComponent{},
			Version:    1,
		}

		// Compile fields
		componentTargets := make(map[string]bool)
		for _, field := range typeDef.Fields {
			compiledField, err := compileField(field)
			if err != nil {
				return nil, fmt.Errorf("failed to compile field '%s' in type '%s': %w", field.Name, typeDef.Name, err)
			}

			for _, sliceType := range compiledField.Slices {
				componentTargets[sliceType.Schema] = true
			}

			cs.Fields = append(cs.Fields, compiledField)
		}

		// Compile slice components
		components, err := compileSliceComponents(componentTargets, typeIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile slice components for type '%s': %w", typeDef.Name, err)
		}
		cs.Components = components

		// Compute checksum
		cs.Checksum = ComputeChecksum(cs)

		compiled = append(compiled, cs)
	}

	return compiled, nil
}

func compileSliceComponents(initialTargets map[string]bool, typeIndex map[string]*TypeDef) ([]CompiledComponent, error) {
	if len(initialTargets) == 0 {
		return []CompiledComponent{}, nil
	}

	queue := make([]string, 0, len(initialTargets))
	for name := range initialTargets {
		queue = append(queue, name)
	}
	sort.Strings(queue)

	visited := make(map[string]bool, len(initialTargets))
	compiledByName := make(map[string]CompiledComponent, len(initialTargets))

	for len(queue) > 0 {
		componentName := queue[0]
		queue = queue[1:]

		if visited[componentName] {
			continue
		}
		visited[componentName] = true

		typeDef, ok := typeIndex[componentName]
		if !ok {
			return nil, fmt.Errorf("slice component type '%s' not found", componentName)
		}

		component := CompiledComponent{
			Name:   componentName,
			Fields: make([]CompiledField, 0, len(typeDef.Fields)),
		}

		for _, field := range typeDef.Fields {
			compiledField, err := compileField(field)
			if err != nil {
				return nil, fmt.Errorf("failed to compile component '%s' field '%s': %w", componentName, field.Name, err)
			}
			component.Fields = append(component.Fields, compiledField)

			for _, sliceType := range compiledField.Slices {
				if !visited[sliceType.Schema] {
					queue = append(queue, sliceType.Schema)
				}
			}
		}

		compiledByName[componentName] = component
	}

	names := make([]string, 0, len(compiledByName))
	for name := range compiledByName {
		names = append(names, name)
	}
	sort.Strings(names)

	components := make([]CompiledComponent, 0, len(names))
	for _, name := range names {
		components = append(components, compiledByName[name])
	}

	return components, nil
}

func compileField(field FieldDef) (CompiledField, error) {
	compiledField := CompiledField{
		Name:       field.Name,
		Type:       field.Type,
		Required:   field.Required,
		Array:      field.Array,
		ArrayReq:   field.ArrayReq,
		Decorators: make(map[string]any),
		InlineEnum: field.InlineEnum,
		IsRelation: field.IsRelation,
		Slices:     []CompiledSliceType{},
	}

	for k, v := range field.Decorators {
		compiledField.Decorators[k] = v
	}

	if field.Required {
		compiledField.Decorators[DecRequired] = true
	}

	if field.IsRelation {
		compiledField.RelationTo = field.Type
	}

	if slicesVal, ok := field.Decorators[DecSlices]; ok {
		slices, err := normalizeSliceDecorator(slicesVal)
		if err != nil {
			return CompiledField{}, err
		}
		compiledField.Slices = slices
	}

	return compiledField, nil
}

func normalizeSliceDecorator(value any) ([]CompiledSliceType, error) {
	mapping, ok := value.(map[string]any)
	if !ok || len(mapping) == 0 {
		return nil, fmt.Errorf("@slices must be a non-empty named mapping")
	}

	result := make([]CompiledSliceType, 0, len(mapping))
	for sliceType, componentAny := range mapping {
		componentName, ok := componentAny.(string)
		if !ok || componentName == "" {
			return nil, fmt.Errorf("slice '%s' must map to a type name", sliceType)
		}

		result = append(result, CompiledSliceType{
			Type:   sliceType,
			Schema: componentName,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Type < result[j].Type
	})

	return result, nil
}

// UpdateVersion increments the version and recomputes checksum
func (cs *CompiledSchema) UpdateVersion() {
	cs.Version++
	cs.Checksum = ComputeChecksum(cs)
}

// HasChanges compares two schemas to detect structural changes
func (cs *CompiledSchema) HasChanges(other *CompiledSchema) bool {
	if other == nil {
		return true
	}

	// Compare checksums (excluding version)
	thisChecksum := ComputeChecksum(cs)
	otherChecksum := ComputeChecksum(other)

	return thisChecksum != otherChecksum
}
