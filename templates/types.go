package templates

// TemplateDefinition represents a complete starter template
type TemplateDefinition struct {
	Name            string                      `json:"name"`
	Slug            string                      `json:"slug"`
	Description     string                      `json:"description"`
	Icon            string                      `json:"icon"`
	Category        string                      `json:"category"`
	Schemas         []SchemaDefinition          `json:"schemas"`
	SampleDocuments map[string][]map[string]any `json:"sample_documents"`
}

// SchemaDefinition represents a schema within a template
type SchemaDefinition struct {
	Name        string `json:"name"`
	ApiID       string `json:"api_id"`
	Icon        string `json:"icon"`
	IsSingleton bool   `json:"is_singleton"`
	FSL         string `json:"fsl"`
}

// PreviewSchema is a minimal schema preview for listing
type PreviewSchema struct {
	Name  string `json:"name"`
	ApiID string `json:"api_id"`
	Icon  string `json:"icon"`
}
