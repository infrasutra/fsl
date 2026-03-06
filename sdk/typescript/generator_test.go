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

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"post", "Post"},
		{"Post", "Post"},
		{"", ""},
		{"blog_post", "BlogPost"},
		{"my-type", "MyType"},
		{"my_blog_post", "MyBlogPost"},
		{"already_PascalCase", "AlreadyPascalCase"},
	}
	for _, tt := range tests {
		got := ToPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateCMSClientPaths(t *testing.T) {
	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{sampleSchema()}, sdk.GeneratorConfig{
		BaseURL:          "",
		WorkspaceID:      "22222222-2222-2222-2222-222222222222",
		ProjectID:        "33333333-3333-3333-3333-333333333333",
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

	expectedList := "/api/v1/cms/projects/${this.projectId}/schemas/11111111-1111-1111-1111-111111111111/documents"
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

func TestGenerateTypedSliceUnions(t *testing.T) {
	gen := New()

	schema := &parser.CompiledSchema{
		Name:     "Page",
		ApiID:    "page",
		SchemaID: "44444444-4444-4444-4444-444444444444",
		Fields: []parser.CompiledField{
			{
				Name:     "title",
				Type:     parser.TypeString,
				Required: true,
			},
			{
				Name:     "slices",
				Type:     parser.TypeJSON,
				Required: true,
				Slices: []parser.CompiledSliceType{
					{Type: "hero", Schema: "HeroSlice"},
					{Type: "faq", Schema: "FaqSlice"},
				},
			},
		},
		Components: []parser.CompiledComponent{
			{
				Name: "HeroSlice",
				Fields: []parser.CompiledField{
					{Name: "heading", Type: parser.TypeString, Required: true},
					{Name: "subheading", Type: parser.TypeText, Required: false},
				},
			},
			{
				Name: "FaqSlice",
				Fields: []parser.CompiledField{
					{Name: "title", Type: parser.TypeString, Required: true},
				},
			},
		},
	}

	generated, err := gen.Generate([]*parser.CompiledSchema{schema}, sdk.GeneratorConfig{
		IncludeClient:    false,
		StrictNullChecks: true,
		TargetAPI:        "cms",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	types, ok := generated.Files["types.ts"]
	if !ok {
		t.Fatalf("types.ts not generated")
	}

	if !strings.Contains(types, "export interface PageHeroSlice") {
		t.Fatalf("types.ts missing generated slice component interface")
	}

	if !strings.Contains(types, "export type PageSlicesSlice =") {
		t.Fatalf("types.ts missing generated slice union type")
	}

	if !strings.Contains(types, `{ type: "hero"; data: PageHeroSlice; variation?: string | null }`) {
		t.Fatalf("types.ts missing hero slice variant with typed data")
	}

	if !strings.Contains(types, "slices: PageSlicesSlice[];") {
		t.Fatalf("types.ts missing typed slice zone field")
	}

	if !strings.Contains(types, "slices?: PageSlicesSlice[] | null;") {
		t.Fatalf("types.ts missing typed create/update input slice zone field")
	}
}
