package python

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
			{
				Name:     "body",
				Type:     parser.TypeText,
				Required: false,
			},
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
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	client, ok := generated.Files["client.py"]
	if !ok {
		t.Fatalf("client.py not generated")
	}

	if !strings.Contains(client, "/api/v1/content/demo-workspace/post") {
		t.Fatalf("client.py missing content list path")
	}

	if !strings.Contains(client, "X-API-Key") {
		t.Fatalf("content client missing X-API-Key header handling")
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
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	models, ok := generated.Files["models.py"]
	if !ok {
		t.Fatalf("models.py not generated")
	}

	if !strings.Contains(models, "class Post(BaseModel):") {
		t.Fatalf("models.py missing Post model")
	}

	if !strings.Contains(models, "title: str") {
		t.Fatalf("models.py missing required field 'title'")
	}

	if !strings.Contains(models, "body: Optional[str]") {
		t.Fatalf("models.py missing optional field 'body'")
	}
}

func TestGenerateEnumLiteral(t *testing.T) {
	schema := &parser.CompiledSchema{
		Name:  "Article",
		ApiID: "article",
		Enums: []parser.CompiledEnum{
			{Name: "Status", Values: []string{"draft", "published", "archived"}},
		},
		Fields: []parser.CompiledField{
			{Name: "title", Type: parser.TypeString, Required: true},
		},
	}

	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{schema}, sdk.GeneratorConfig{
		IncludeClient: false,
		TargetAPI:     "content",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	models := generated.Files["models.py"]
	if !strings.Contains(models, `Status = Literal["draft", "published", "archived"]`) {
		t.Fatalf("models.py missing Literal enum type alias")
	}
}

func TestGenerateCMSClientMethods(t *testing.T) {
	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{sampleSchema()}, sdk.GeneratorConfig{
		BaseURL:       "",
		IncludeClient: true,
		TargetAPI:     "cms",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	client, ok := generated.Files["client.py"]
	if !ok {
		t.Fatalf("client.py not generated")
	}

	checks := []string{
		"def list_post(self, project_id: str",
		"def get_post(self, project_id: str, document_id: str",
		"def create_post(self, project_id: str, input: CreatePostInput",
		"def update_post(self, project_id: str, document_id: str, input: UpdatePostInput",
		"def delete_post(self, project_id: str, document_id: str",
		"/api/v1/projects/{project_id}/documents",
		`"schema_api_id": "post"`,
		"Authorization",
	}
	for _, check := range checks {
		if !strings.Contains(client, check) {
			t.Fatalf("client.py missing expected content: %q", check)
		}
	}

	if strings.Contains(client, "X-API-Key") {
		t.Fatalf("cms client should not use X-API-Key header")
	}
}

func TestGenerateCreateUpdateInputs(t *testing.T) {
	gen := New()
	generated, err := gen.Generate([]*parser.CompiledSchema{sampleSchema()}, sdk.GeneratorConfig{
		IncludeClient: false,
		TargetAPI:     "content",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	models := generated.Files["models.py"]

	if !strings.Contains(models, "class CreatePostInput(BaseModel):") {
		t.Fatalf("models.py missing CreatePostInput")
	}

	if !strings.Contains(models, "class UpdatePostInput(BaseModel):") {
		t.Fatalf("models.py missing UpdatePostInput")
	}
}
