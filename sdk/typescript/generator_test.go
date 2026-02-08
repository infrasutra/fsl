package typescript

import (
	"strings"
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/infrasutra/fsl/sdk"
)

func sampleSchema() *parser.CompiledSchema {
	return &parser.CompiledSchema{
		Name:     "Post",
		ApiID:    "post",
		SchemaID: "11111111-1111-1111-1111-111111111111",
		Fields: []parser.CompiledField{
			{
				Name:     "title",
				Type:     parser.TypeString,
				Required: true,
			},
		},
	}
}

func TestGenerateCMSClientPaths(t *testing.T) {
	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{sampleSchema()}, sdk.GeneratorConfig{
		BaseURL:          "",
		WorkspaceID:      "22222222-2222-2222-2222-222222222222",
		IncludeClient:    true,
		StrictNullChecks: true,
		TargetAPI:        "cms",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	client, ok := generated.Files["client.ts"]
	if !ok {
		t.Fatalf("client.ts not generated")
	}

	expectedList := "/api/v1/cms/workspaces/${this.workspaceId}/schemas/11111111-1111-1111-1111-111111111111/documents"
	if !strings.Contains(client, expectedList) {
		t.Fatalf("client.ts missing CMS list path: %s", expectedList)
	}

	if !strings.Contains(client, "Authorization") {
		t.Fatalf("client.ts missing Authorization header handling")
	}

	types, ok := generated.Files["types.ts"]
	if !ok {
		t.Fatalf("types.ts not generated")
	}

	if !strings.Contains(types, "export interface Post") {
		t.Fatalf("types.ts missing Post interface")
	}

	if strings.Contains(types, "createdAt:") {
		t.Fatalf("types.ts should not include camelCase system fields")
	}
}

func TestGenerateContentClientPaths(t *testing.T) {
	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{sampleSchema()}, sdk.GeneratorConfig{
		BaseURL:          "",
		WorkspaceAPIID:   "demo-workspace",
		IncludeClient:    true,
		StrictNullChecks: true,
		TargetAPI:        "content",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	client, ok := generated.Files["client.ts"]
	if !ok {
		t.Fatalf("client.ts not generated")
	}

	expectedList := "/api/v1/content/${this.workspaceApiId}/post"
	if !strings.Contains(client, expectedList) {
		t.Fatalf("client.ts missing content list path: %s", expectedList)
	}

	if !strings.Contains(client, "/api/v1/content/${this.workspaceApiId}/post/${slug}") {
		t.Fatalf("client.ts missing content getBySlug path")
	}

	if !strings.Contains(client, "X-API-Key") {
		t.Fatalf("content client missing X-API-Key header handling")
	}

	if strings.Contains(client, "Authorization") {
		t.Fatalf("content client should not include Authorization header")
	}
}
