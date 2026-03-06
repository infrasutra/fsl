package python

import (
	"testing"

	"github.com/infrasutra/fsl/parser"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"camelCase", "camel_case"},
		{"HTMLParser", "html_parser"},
		{"simpleID", "simple_id"},
		{"", ""},
		{"Post", "post"},
		{"alreadysnake", "alreadysnake"},
		{"APIKeyID", "api_key_id"},
	}
	for _, tt := range tests {
		got := ToSnakeCase(tt.input)
		if got != tt.want {
			t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
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
		{"alreadyPascal", "AlreadyPascal"},
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

func TestMapFieldType(t *testing.T) {
	tests := []struct {
		name  string
		field *parser.CompiledField
		want  string
	}{
		{
			name:  "required string",
			field: &parser.CompiledField{Name: "title", Type: parser.TypeString, Required: true},
			want:  "str",
		},
		{
			name:  "optional string",
			field: &parser.CompiledField{Name: "bio", Type: parser.TypeString, Required: false},
			want:  "str",
		},
		{
			name:  "relation",
			field: &parser.CompiledField{Name: "author", Type: "Author", IsRelation: true, RelationTo: "Author"},
			want:  "Author",
		},
		{
			name:  "array relation",
			field: &parser.CompiledField{Name: "tags", Type: "Tag", IsRelation: true, RelationTo: "Tag", Array: true},
			want:  "list[Tag]",
		},
		{
			name:  "inline enum",
			field: &parser.CompiledField{Name: "status", Type: "Enum", InlineEnum: []string{"draft", "published"}},
			want:  `Literal["draft", "published"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapFieldType(tt.field)
			if got != tt.want {
				t.Errorf("MapFieldType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapFieldTypeWithOptional(t *testing.T) {
	tests := []struct {
		name  string
		field *parser.CompiledField
		want  string
	}{
		{
			name:  "required string",
			field: &parser.CompiledField{Name: "title", Type: parser.TypeString, Required: true},
			want:  "str",
		},
		{
			name:  "optional string",
			field: &parser.CompiledField{Name: "bio", Type: parser.TypeString, Required: false},
			want:  "Optional[str]",
		},
		{
			name:  "required array",
			field: &parser.CompiledField{Name: "tags", Type: parser.TypeString, Array: true, ArrayReq: true},
			want:  "list[str]",
		},
		{
			name:  "optional array",
			field: &parser.CompiledField{Name: "tags", Type: parser.TypeString, Array: true},
			want:  "Optional[list[str]]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapFieldTypeWithOptional(tt.field)
			if got != tt.want {
				t.Errorf("MapFieldTypeWithOptional() = %q, want %q", got, tt.want)
			}
		})
	}
}
