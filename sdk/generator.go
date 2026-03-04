// Package sdk provides SDK code generation from FSL schemas
package sdk

import (
	"github.com/infrasutra/fsl/parser"
)

// Generator defines the interface for SDK code generators
type Generator interface {
	// Generate produces SDK code from compiled schemas
	Generate(schemas []*parser.CompiledSchema, config GeneratorConfig) (*GeneratedSDK, error)

	// Language returns the target language name
	Language() string

	// FileExtension returns the file extension for generated files
	FileExtension() string
}

// GeneratorConfig holds configuration options for code generation
type GeneratorConfig struct {
	// PackageName is the name for the generated package/module
	PackageName string

	// TargetAPI selects the API type ("cms" or "content")
	TargetAPI string

	// BaseURL is the API base URL to embed in the client
	BaseURL string

	// WorkspaceID is the workspace identifier
	WorkspaceID string

	// ProjectID is the project identifier for CMS APIs
	ProjectID string

	// WorkspaceAPIID is the workspace API identifier
	WorkspaceAPIID string

	// IncludeClient generates a full API client, not just types
	IncludeClient bool

	// IncludeValidation generates runtime validation helpers
	IncludeValidation bool

	// StrictNullChecks enables strict null checking in TypeScript
	StrictNullChecks bool
}

// GeneratedSDK represents the output of code generation
type GeneratedSDK struct {
	// Language is the target language
	Language string

	// Files is a map of filename to content
	Files map[string]string

	// EntryPoint is the main file to import
	EntryPoint string
}

// GetFile returns a specific file content
func (sdk *GeneratedSDK) GetFile(name string) (string, bool) {
	content, ok := sdk.Files[name]
	return content, ok
}

// FileList returns all generated filenames
func (sdk *GeneratedSDK) FileList() []string {
	files := make([]string, 0, len(sdk.Files))
	for name := range sdk.Files {
		files = append(files, name)
	}
	return files
}
