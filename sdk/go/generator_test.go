package gosdk

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
		Enums: []parser.CompiledEnum{
			{Name: "PostStatus", Values: []string{"draft", "published"}},
		},
		Fields: []parser.CompiledField{
			{Name: "title", Type: parser.TypeString, Required: true},
			{Name: "body", Type: parser.TypeText, Required: false},
			{Name: "status", Type: "PostStatus", Required: true},
			{Name: "author", Type: "Author", IsRelation: true, RelationTo: "Author", Required: false},
		},
	}
}

func TestGenerateContentClientPaths(t *testing.T) {
	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{sampleSchema()}, sdk.GeneratorConfig{
		BaseURL:        "",
		WorkspaceAPIID: "demo-workspace",
		IncludeClient:  true,
		TargetAPI:      "content",
		PackageName:    "client",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	client, ok := generated.Files["client.go"]
	if !ok {
		t.Fatalf("client.go not generated")
	}

	checks := []string{
		"func NewFluxClient(config ClientConfig, workspaceAPIID string) *FluxClient",
		"headers[\"X-API-Key\"] = config.APIKey",
		"/api/v1/content/%s/post",
		"func (c *FluxClient) GetPostBySlug(ctx context.Context, slug string)",
		"func (c *FluxClient) GetPostByID(ctx context.Context, id string)",
	}

	for _, check := range checks {
		if !strings.Contains(client, check) {
			t.Fatalf("client.go missing expected content: %q", check)
		}
	}

	if strings.Contains(client, "Authorization") {
		t.Fatalf("content client should not include Authorization header")
	}
}

func TestGenerateModels(t *testing.T) {
	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{sampleSchema()}, sdk.GeneratorConfig{
		IncludeClient: false,
		TargetAPI:     "content",
		PackageName:   "client",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	models, ok := generated.Files["models.go"]
	if !ok {
		t.Fatalf("models.go not generated")
	}

	checks := []string{
		"type Post struct {",
		"Title string `json:\"title\"`",
		"Body *string `json:\"body,omitempty\"`",
		"Status PostStatus `json:\"status\"`",
		"Author *Author `json:\"author,omitempty\"`",
		"type CreatePostInput struct {",
		"type UpdatePostInput struct {",
		"type PostFilter struct {",
		"type PostStatus string",
		"type PostDocument = DocumentResponse[Post]",
	}

	for _, check := range checks {
		if !strings.Contains(models, check) {
			t.Fatalf("models.go missing expected content: %q", check)
		}
	}
}

func TestGenerateTypedSlices(t *testing.T) {
	gen := New()

	schema := &parser.CompiledSchema{
		Name:  "Page",
		ApiID: "page",
		Fields: []parser.CompiledField{
			{Name: "title", Type: parser.TypeString, Required: true},
			{
				Name:     "slices",
				Type:     parser.TypeJSON,
				Required: true,
				Slices: []parser.CompiledSliceType{
					{Type: "hero", Schema: "HeroSlice"},
				},
			},
		},
		Components: []parser.CompiledComponent{
			{
				Name: "HeroSlice",
				Fields: []parser.CompiledField{
					{Name: "heading", Type: parser.TypeString, Required: true},
				},
			},
		},
	}

	generated, err := gen.Generate([]*parser.CompiledSchema{schema}, sdk.GeneratorConfig{
		IncludeClient: false,
		TargetAPI:     "content",
		PackageName:   "client",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	models := generated.Files["models.go"]

	checks := []string{
		"type PageHeroSlice struct {",
		"type PageSlicesSlice struct {",
		"Slices []PageSlicesSlice `json:\"slices\"`",
	}

	for _, check := range checks {
		if !strings.Contains(models, check) {
			t.Fatalf("models.go missing expected content: %q", check)
		}
	}
}
