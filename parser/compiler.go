package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// CompiledSchema is the output format for storage
type CompiledSchema struct {
	Name        string             `json:"name"`
	SchemaID    string             `json:"schemaId,omitempty"`
	ApiID       string             `json:"apiId"`
	Singleton   bool               `json:"singleton"`
	Collection  string             `json:"collection,omitempty"`  // Custom collection name from @collection
	Icon        string             `json:"icon,omitempty"`        // Lucide icon name from @icon
	Description string             `json:"description,omitempty"` // Schema description from @description
	Fields      []CompiledField    `json:"fields"`
	Relations   []CompiledRelation `json:"relations,omitempty"` // Phase 2: relation metadata
	Enums       []CompiledEnum     `json:"enums,omitempty"`     // Phase 2: named enums used
	Version     int                `json:"version"`
	Checksum    string             `json:"checksum"`
}

type CompiledField struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Required   bool           `json:"required"`
	Array      bool           `json:"array"`
	ArrayReq   bool           `json:"arrayRequired,omitempty"`
	Decorators map[string]any `json:"decorators,omitempty"`

	// Phase 2 fields
	InlineEnum []string `json:"inlineEnum,omitempty"` // Inline enum values
	IsRelation bool     `json:"isRelation,omitempty"` // True if field is a relation
	RelationTo string   `json:"relationTo,omitempty"` // Target type for relations
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
		Name:      name,
		ApiID:     apiID,
		Singleton: singleton,
		Fields:    make([]CompiledField, 0, len(typeDef.Fields)),
		Relations: []CompiledRelation{},
		Enums:     []CompiledEnum{},
		Version:   1,
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
	for _, field := range typeDef.Fields {
		compiledField := CompiledField{
			Name:       field.Name,
			Type:       field.Type,
			Required:   field.Required,
			Array:      field.Array,
			ArrayReq:   field.ArrayReq,
			Decorators: make(map[string]any),
			InlineEnum: field.InlineEnum,
			IsRelation: field.IsRelation,
		}

		// Copy decorators
		for k, v := range field.Decorators {
			compiledField.Decorators[k] = v
		}

		// Add implicit required decorator if field is required
		if field.Required {
			compiledField.Decorators[DecRequired] = true
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

	// Compute checksum
	compiled.Checksum = ComputeChecksum(compiled)

	return compiled, nil
}

// ComputeChecksum generates SHA-256 checksum of the compiled schema
// Excludes version and checksum fields from the hash
func ComputeChecksum(cs *CompiledSchema) string {
	// Create a copy without version and checksum for hashing
	hashData := struct {
		Name        string             `json:"name"`
		ApiID       string             `json:"apiId"`
		Singleton   bool               `json:"singleton"`
		Collection  string             `json:"collection,omitempty"`
		Icon        string             `json:"icon,omitempty"`
		Description string             `json:"description,omitempty"`
		Fields      []CompiledField    `json:"fields"`
		Relations   []CompiledRelation `json:"relations,omitempty"`
		Enums       []CompiledEnum     `json:"enums,omitempty"`
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

	for i, typeDef := range schema.Types {
		// Generate unique API ID for each type
		apiID := fmt.Sprintf("%s_%d", baseApiID, i)
		if baseApiID == "" {
			apiID = fmt.Sprintf("type_%d", i)
		}

		cs := &CompiledSchema{
			Name:      typeDef.Name,
			ApiID:     apiID,
			Singleton: singleton,
			Fields:    make([]CompiledField, 0, len(typeDef.Fields)),
			Version:   1,
		}

		// Compile fields
		for _, field := range typeDef.Fields {
			compiledField := CompiledField{
				Name:       field.Name,
				Type:       field.Type,
				Required:   field.Required,
				Array:      field.Array,
				ArrayReq:   field.ArrayReq,
				Decorators: make(map[string]any),
			}

			// Copy decorators
			for k, v := range field.Decorators {
				compiledField.Decorators[k] = v
			}

			// Add implicit required decorator if field is required
			if field.Required {
				compiledField.Decorators[DecRequired] = true
			}

			cs.Fields = append(cs.Fields, compiledField)
		}

		// Compute checksum
		cs.Checksum = ComputeChecksum(cs)

		compiled = append(compiled, cs)
	}

	return compiled, nil
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
