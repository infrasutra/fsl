package openapi

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/infrasutra/fsl/parser"
	"github.com/infrasutra/fsl/sdk"
)

const (
	formatOpenAPI    = "openapi"
	formatJSONSchema = "jsonschema"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) Language() string {
	return "openapi"
}

func (g *Generator) FileExtension() string {
	return ".json"
}

func (g *Generator) Generate(schemas []*parser.CompiledSchema, config sdk.GeneratorConfig) (*sdk.GeneratedSDK, error) {
	format := normalizeFormat(config.ExportFormat)

	var (
		doc      map[string]any
		fileName string
	)

	switch format {
	case formatOpenAPI:
		doc = g.generateOpenAPI(schemas)
		fileName = "openapi.json"
	case formatJSONSchema:
		doc = g.generateJSONSchema(schemas)
		fileName = "jsonschema.json"
	default:
		return nil, fmt.Errorf("unsupported export format %q: must be openapi or jsonschema", format)
	}

	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s document: %w", format, err)
	}

	return &sdk.GeneratedSDK{
		Language:   format,
		EntryPoint: fileName,
		Files: map[string]string{
			fileName: string(payload) + "\n",
		},
	}, nil
}

func normalizeFormat(format string) string {
	if format == "" {
		return formatOpenAPI
	}
	return strings.ToLower(strings.TrimSpace(format))
}

func (g *Generator) generateOpenAPI(schemas []*parser.CompiledSchema) map[string]any {
	components := g.buildDefinitionMap(schemas, true)

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Flux CMS Compiled Schemas",
			"version": "1.0.0",
		},
		"paths": map[string]any{},
		"components": map[string]any{
			"schemas": components,
		},
	}
}

func (g *Generator) generateJSONSchema(schemas []*parser.CompiledSchema) map[string]any {
	defs := g.buildDefinitionMap(schemas, false)
	properties := map[string]any{}

	for _, schema := range schemas {
		defName := schemaDefName(schema.Name)
		properties[defName] = map[string]any{"$ref": "#/$defs/" + defName}
	}

	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"title":                "Flux CMS Compiled Schemas",
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"$defs":                defs,
	}
}

func (g *Generator) buildDefinitionMap(schemas []*parser.CompiledSchema, openapi bool) map[string]any {
	definitions := map[string]any{}
	enums := map[string][]string{}

	for _, schema := range schemas {
		for _, enumDef := range schema.Enums {
			if _, exists := enums[enumDef.Name]; !exists {
				enums[enumDef.Name] = enumDef.Values
			}
		}
	}

	for enumName, values := range enums {
		definitions[enumName] = map[string]any{
			"type": "string",
			"enum": values,
		}
	}

	for _, schema := range schemas {
		for _, component := range schema.Components {
			definitions[componentDefName(schema.Name, component.Name)] = g.objectSchema(schema.Name, component.Fields, enums, openapi)
		}
	}

	for _, schema := range schemas {
		schemaObj := g.objectSchema(schema.Name, schema.Fields, enums, openapi)
		schemaObj["title"] = schema.Name
		if schema.Description != "" {
			schemaObj["description"] = schema.Description
		}
		schemaObj["x-fsl-apiId"] = schema.ApiID
		schemaObj["x-fsl-singleton"] = schema.Singleton
		if schema.Collection != "" {
			schemaObj["x-fsl-collection"] = schema.Collection
		}
		if schema.Icon != "" {
			schemaObj["x-fsl-icon"] = schema.Icon
		}
		if len(schema.Relations) > 0 {
			schemaObj["x-fsl-relations"] = schema.Relations
		}
		definitions[schemaDefName(schema.Name)] = schemaObj
	}

	return definitions
}

func (g *Generator) objectSchema(parentSchemaName string, fields []parser.CompiledField, enums map[string][]string, openapi bool) map[string]any {
	properties := map[string]any{}
	required := make([]string, 0)

	for i := range fields {
		field := fields[i]
		fieldSchema := g.fieldSchema(parentSchemaName, &field, enums, openapi)
		properties[field.Name] = fieldSchema

		if field.Required || (field.Array && field.ArrayReq) {
			required = append(required, field.Name)
		}
	}

	obj := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}

	if len(required) > 0 {
		obj["required"] = required
	}

	return obj
}

func (g *Generator) fieldSchema(parentSchemaName string, field *parser.CompiledField, enums map[string][]string, openapi bool) map[string]any {
	var base map[string]any

	switch {
	case len(field.Slices) > 0:
		base = g.sliceZoneSchema(parentSchemaName, field, openapi)
	case field.IsRelation:
		base = relationSchema(openapi)
	case field.Type == "Enum" && len(field.InlineEnum) > 0:
		base = map[string]any{"type": "string", "enum": field.InlineEnum}
	case enums[field.Type] != nil:
		base = g.refSchema(field.Type, openapi)
	default:
		base = g.primitiveSchema(field, openapi)
	}

	if field.Array {
		arraySchema := map[string]any{
			"type":  "array",
			"items": base,
		}
		if minItems, ok := toInt(field.Decorators[parser.DecMinItems]); ok {
			arraySchema["minItems"] = minItems
		}
		if maxItems, ok := toInt(field.Decorators[parser.DecMaxItems]); ok {
			arraySchema["maxItems"] = maxItems
		}
		if field.ArrayReq {
			if existing, ok := arraySchema["minItems"].(int64); !ok || existing < 1 {
				arraySchema["minItems"] = int64(1)
			}
		}
		base = arraySchema
	}

	g.applyDecorators(base, field)

	if !field.Required && !field.ArrayReq {
		base = nullableSchema(base, openapi)
	}

	return base
}

func (g *Generator) primitiveSchema(field *parser.CompiledField, openapi bool) map[string]any {
	switch field.Type {
	case parser.TypeString, parser.TypeText:
		return map[string]any{"type": "string"}
	case parser.TypeInt:
		return map[string]any{"type": "integer"}
	case parser.TypeFloat:
		return map[string]any{"type": "number"}
	case parser.TypeBoolean:
		return map[string]any{"type": "boolean"}
	case parser.TypeDateTime:
		return map[string]any{"type": "string", "format": "date-time"}
	case parser.TypeDate:
		return map[string]any{"type": "string", "format": "date"}
	case parser.TypeRichText:
		return map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"type": map[string]any{"type": "string"}},
				"required":             []string{"type"},
				"additionalProperties": true,
			},
		}
	case parser.TypeImage:
		return imageAssetSchema()
	case parser.TypeFile:
		return fileAssetSchema()
	case parser.TypeJSON:
		return map[string]any{}
	default:
		return map[string]any{"type": "object"}
	}
}

func (g *Generator) sliceZoneSchema(parentSchemaName string, field *parser.CompiledField, openapi bool) map[string]any {
	oneOf := make([]any, 0, len(field.Slices))

	for _, slice := range field.Slices {
		variant := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type": "string",
					"enum": []string{slice.Type},
				},
				"data": g.refSchema(componentDefName(parentSchemaName, slice.Schema), openapi),
			},
			"required":             []string{"type", "data"},
			"additionalProperties": false,
		}
		oneOf = append(oneOf, variant)
	}

	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"oneOf": oneOf,
		},
	}
}

func relationSchema(openapi bool) map[string]any {
	refObject := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":   "string",
				"format": "uuid",
			},
		},
		"required":             []string{"id"},
		"additionalProperties": false,
	}

	if openapi {
		return map[string]any{
			"oneOf": []any{
				map[string]any{"type": "string", "format": "uuid"},
				refObject,
			},
		}
	}

	return map[string]any{
		"oneOf": []any{
			map[string]any{"type": "string", "format": "uuid"},
			refObject,
		},
	}
}

func (g *Generator) refSchema(name string, openapi bool) map[string]any {
	if openapi {
		return map[string]any{"$ref": "#/components/schemas/" + name}
	}
	return map[string]any{"$ref": "#/$defs/" + name}
}

func nullableSchema(base map[string]any, openapi bool) map[string]any {
	if openapi {
		base["nullable"] = true
		return base
	}

	return map[string]any{
		"anyOf": []any{
			base,
			map[string]any{"type": "null"},
		},
	}
}

func (g *Generator) applyDecorators(schema map[string]any, field *parser.CompiledField) {
	if maxLen, ok := toInt(field.Decorators[parser.DecMaxLength]); ok {
		schema["maxLength"] = maxLen
	}
	if minLen, ok := toInt(field.Decorators[parser.DecMinLength]); ok {
		schema["minLength"] = minLen
	}
	if pattern, ok := field.Decorators[parser.DecPattern].(string); ok && pattern != "" {
		schema["pattern"] = pattern
	}
	if minValue, ok := toFloat(field.Decorators[parser.DecMin]); ok {
		schema["minimum"] = minValue
	}
	if maxValue, ok := toFloat(field.Decorators[parser.DecMax]); ok {
		schema["maximum"] = maxValue
	}
	if precision, ok := toInt(field.Decorators[parser.DecPrecision]); ok && precision >= 0 {
		schema["multipleOf"] = math.Pow10(-int(precision))
	}
	if defaultValue, ok := field.Decorators[parser.DecDefault]; ok {
		schema["default"] = defaultValue
	}

	if formats, ok := normalizeStringList(field.Decorators[parser.DecFormats]); ok {
		schema["x-fsl-formats"] = formats
	}
	if maxSize, ok := toInt(field.Decorators[parser.DecMaxSize]); ok {
		schema["x-fsl-maxSize"] = maxSize
	}
	if hidden, ok := field.Decorators[parser.DecHidden].(bool); ok && hidden {
		schema["writeOnly"] = true
		schema["x-fsl-hidden"] = true
	}
	if label, ok := field.Decorators[parser.DecLabel].(string); ok && label != "" {
		schema["x-fsl-label"] = label
	}
	if help, ok := field.Decorators[parser.DecHelp].(string); ok && help != "" {
		schema["x-fsl-help"] = help
	}
	if placeholder, ok := field.Decorators[parser.DecPlaceholder].(string); ok && placeholder != "" {
		schema["x-fsl-placeholder"] = placeholder
	}
	if unique, ok := field.Decorators[parser.DecUnique].(bool); ok {
		schema["x-fsl-unique"] = unique
	}
	if index, ok := field.Decorators[parser.DecIndex].(bool); ok {
		schema["x-fsl-index"] = index
	}
	if searchable, ok := field.Decorators[parser.DecSearchable].(bool); ok {
		schema["x-fsl-searchable"] = searchable
	}
	if relation, ok := field.Decorators[parser.DecRelation].(map[string]any); ok {
		schema["x-fsl-relation"] = relation
	}
}

func imageAssetSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":      map[string]any{"type": "string"},
			"width":    map[string]any{"type": "integer"},
			"height":   map[string]any{"type": "integer"},
			"alt":      map[string]any{"type": "string"},
			"filename": map[string]any{"type": "string"},
			"size":     map[string]any{"type": "integer"},
			"mimeType": map[string]any{"type": "string"},
		},
		"required": []string{"url"},
	}
}

func fileAssetSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":      map[string]any{"type": "string"},
			"filename": map[string]any{"type": "string"},
			"size":     map[string]any{"type": "integer"},
			"mimeType": map[string]any{"type": "string"},
		},
		"required": []string{"url"},
	}
}

func schemaDefName(name string) string {
	return toPascal(name)
}

func componentDefName(schemaName, componentName string) string {
	if schemaName == "" {
		return toPascal(componentName)
	}
	return toPascal(schemaName) + toPascal(componentName)
}

func toPascal(s string) string {
	if s == "" {
		return s
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return s
	}

	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		if runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] = runes[0] - ('a' - 'A')
		}
		b.WriteString(string(runes))
	}
	return b.String()
}

func toInt(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func normalizeStringList(value any) ([]string, bool) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, false
		}
		return []string{v}, true
	case []string:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	case []any:
		if len(v) == 0 {
			return nil, false
		}
		result := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if !ok || str == "" {
				continue
			}
			result = append(result, str)
		}
		if len(result) == 0 {
			return nil, false
		}
		return result, true
	default:
		return nil, false
	}
}
