// Package template provides types and utilities for schema template files
package template

// TemplateFile represents a schema template in file format (YAML/JSON/FSL)
type TemplateFile struct {
	// Version of the template file format (currently "1")
	Version string `json:"version" yaml:"version"`

	// Name is the display name of the template
	Name string `json:"name" yaml:"name"`

	// Description provides additional context about the template
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Icon is the Lucide icon name for the template
	Icon string `json:"icon,omitempty" yaml:"icon,omitempty"`

	// Category classifies the template (content, commerce, marketing, system, custom)
	Category string `json:"category" yaml:"category"`

	// IsSingleton indicates if schemas created from this template should be singletons
	IsSingleton bool `json:"is_singleton,omitempty" yaml:"is_singleton,omitempty"`

	// Tags are searchable keywords for the template
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// FSL is the Flux Schema Language definition
	FSL string `json:"fsl" yaml:"fsl"`
}

// TemplateCategory represents valid template categories
type TemplateCategory string

const (
	CategoryContent   TemplateCategory = "content"
	CategoryCommerce  TemplateCategory = "commerce"
	CategoryMarketing TemplateCategory = "marketing"
	CategorySystem    TemplateCategory = "system"
	CategoryCustom    TemplateCategory = "custom"
)

// ValidCategories returns all valid category values
func ValidCategories() []TemplateCategory {
	return []TemplateCategory{
		CategoryContent,
		CategoryCommerce,
		CategoryMarketing,
		CategorySystem,
		CategoryCustom,
	}
}

// IsValidCategory checks if a category string is valid
func IsValidCategory(cat string) bool {
	for _, valid := range ValidCategories() {
		if string(valid) == cat {
			return true
		}
	}
	return false
}

// TemplateSource represents the origin of a template
type TemplateSource string

const (
	SourceBuiltin  TemplateSource = "builtin"
	SourceUser     TemplateSource = "user"
	SourceImported TemplateSource = "imported"
)
