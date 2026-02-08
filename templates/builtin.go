package templates

import "encoding/json"

// GetBuiltinTemplates returns all built-in template definitions
func GetBuiltinTemplates() []TemplateDefinition {
	return []TemplateDefinition{
		BlogTemplate,
		NewsPortalTemplate,
		EcommerceTemplate,
		PortfolioTemplate,
	}
}

// GetTemplateBySlug returns a template by its slug
func GetTemplateBySlug(slug string) *TemplateDefinition {
	for _, t := range GetBuiltinTemplates() {
		if t.Slug == slug {
			return &t
		}
	}
	return nil
}

// GetPreviewSchemas extracts preview schemas from a definition
func (t *TemplateDefinition) GetPreviewSchemas() []PreviewSchema {
	previews := make([]PreviewSchema, len(t.Schemas))
	for i, s := range t.Schemas {
		previews[i] = PreviewSchema{
			Name:  s.Name,
			ApiID: s.ApiID,
			Icon:  s.Icon,
		}
	}
	return previews
}

// ToJSON converts the definition to JSON bytes
func (t *TemplateDefinition) ToJSON() ([]byte, error) {
	return json.Marshal(struct {
		Schemas         []SchemaDefinition          `json:"schemas"`
		SampleDocuments map[string][]map[string]any `json:"sample_documents"`
	}{
		Schemas:         t.Schemas,
		SampleDocuments: t.SampleDocuments,
	})
}
