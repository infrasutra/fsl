package openapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/infrasutra/fsl/sdk"
)

func compiledSchemaFixture() *parser.CompiledSchema {
	return &parser.CompiledSchema{
		Name:        "Post",
		ApiID:       "post",
		Description: "Post content model",
		Fields: []parser.CompiledField{
			{
				Name:     "title",
				Type:     parser.TypeString,
				Required: true,
				Decorators: map[string]any{
					parser.DecMaxLength: 120,
					parser.DecPattern:   "^[a-zA-Z0-9 ]+$",
				},
			},
			{
				Name:     "rating",
				Type:     parser.TypeFloat,
				Required: false,
				Decorators: map[string]any{
					parser.DecMin:       0,
					parser.DecMax:       5,
					parser.DecPrecision: 2,
				},
			},
			{
				Name:     "status",
				Type:     "Status",
				Required: true,
			},
			{
				Name:       "author",
				Type:       "Author",
				IsRelation: true,
				Required:   true,
				RelationTo: "Author",
				Decorators: map[string]any{
					parser.DecRelation: map[string]any{"inverse": "posts", "onDelete": "setNull"},
				},
			},
			{
				Name:  "slices",
				Type:  parser.TypeJSON,
				Array: false,
				Slices: []parser.CompiledSliceType{
					{Type: "hero", Schema: "HeroSlice"},
				},
			},
			{
				Name:  "gallery",
				Type:  parser.TypeImage,
				Array: true,
				Decorators: map[string]any{
					parser.DecMinItems: 1,
					parser.DecFormats:  []any{"jpg", "png"},
				},
			},
		},
		Enums: []parser.CompiledEnum{
			{Name: "Status", Values: []string{"draft", "published"}},
		},
		Components: []parser.CompiledComponent{
			{
				Name: "HeroSlice",
				Fields: []parser.CompiledField{
					{Name: "headline", Type: parser.TypeString, Required: true},
				},
			},
		},
	}
}

func TestGenerateOpenAPIExport(t *testing.T) {
	gen := New()
	output, err := gen.Generate([]*parser.CompiledSchema{compiledSchemaFixture()}, sdk.GeneratorConfig{ExportFormat: "openapi"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content, ok := output.Files["openapi.json"]
	if !ok {
		t.Fatalf("expected openapi.json output file")
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("generated content is not valid JSON: %v", err)
	}

	if got, _ := doc["openapi"].(string); got != "3.0.3" {
		t.Fatalf("expected OpenAPI version 3.0.3, got %q", got)
	}

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	post := schemas["Post"].(map[string]any)
	props := post["properties"].(map[string]any)

	title := props["title"].(map[string]any)
	if title["maxLength"].(float64) != 120 {
		t.Fatalf("expected maxLength decorator to map to schema")
	}

	status := props["status"].(map[string]any)
	if ref := status["$ref"].(string); ref != "#/components/schemas/Status" {
		t.Fatalf("expected named enum ref, got %q", ref)
	}

	author := props["author"].(map[string]any)
	if _, ok := author["oneOf"]; !ok {
		t.Fatalf("expected relation field to generate oneOf schema")
	}

	gallery := props["gallery"].(map[string]any)
	if gallery["minItems"].(float64) != 1 {
		t.Fatalf("expected minItems decorator to map to schema")
	}
	if _, ok := gallery["x-fsl-formats"]; !ok {
		t.Fatalf("expected x-fsl-formats metadata")
	}
}

func TestGenerateJSONSchemaExport(t *testing.T) {
	gen := New()
	output, err := gen.Generate([]*parser.CompiledSchema{compiledSchemaFixture()}, sdk.GeneratorConfig{ExportFormat: "jsonschema"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content, ok := output.Files["jsonschema.json"]
	if !ok {
		t.Fatalf("expected jsonschema.json output file")
	}

	if !strings.Contains(content, `"$schema": "https://json-schema.org/draft/2020-12/schema"`) {
		t.Fatalf("expected generated JSON Schema document")
	}
	if !strings.Contains(content, `"$defs"`) {
		t.Fatalf("expected $defs in JSON Schema output")
	}
	if !strings.Contains(content, `"anyOf"`) {
		t.Fatalf("expected nullable optional fields to use anyOf in JSON Schema")
	}
}
