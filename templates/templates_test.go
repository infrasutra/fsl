package templates

import (
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBuiltinTemplates(t *testing.T) {
	templates := GetBuiltinTemplates()
	require.NotEmpty(t, templates)
	assert.GreaterOrEqual(t, len(templates), 4, "should have at least 4 built-in templates")

	t.Run("all templates have required fields", func(t *testing.T) {
		for _, tmpl := range templates {
			assert.NotEmpty(t, tmpl.Name, "template name required")
			assert.NotEmpty(t, tmpl.Slug, "template slug required")
			assert.NotEmpty(t, tmpl.Description, "template description required")
			assert.NotEmpty(t, tmpl.Icon, "template icon required")
			assert.NotEmpty(t, tmpl.Category, "template category required")
			assert.NotEmpty(t, tmpl.Schemas, "template should have at least one schema")
		}
	})

	t.Run("all template schemas have valid FSL with cross-schema types", func(t *testing.T) {
		for _, tmpl := range templates {
			// Collect all type names across all schemas in this template
			// (templates can have cross-schema relations)
			var externalTypes []string
			for _, s := range tmpl.Schemas {
				result := parser.ParseWithDiagnostics(s.FSL)
				if result.Schema != nil {
					for _, td := range result.Schema.Types {
						externalTypes = append(externalTypes, td.Name)
					}
					for _, ed := range result.Schema.Enums {
						externalTypes = append(externalTypes, ed.Name)
					}
				}
			}

			// Now validate each schema with external types
			for _, schema := range tmpl.Schemas {
				assert.NotEmpty(t, schema.Name, "schema name required in %s", tmpl.Slug)
				assert.NotEmpty(t, schema.ApiID, "schema api_id required in %s", tmpl.Slug)
				assert.NotEmpty(t, schema.FSL, "schema FSL required in %s/%s", tmpl.Slug, schema.Name)

				_, err := parser.ParseAndCompileWithExternalTypes(
					schema.FSL, schema.Name, schema.ApiID, schema.IsSingleton, externalTypes,
				)
				if err != nil {
					t.Errorf("FSL compile error in %s/%s: %v", tmpl.Slug, schema.Name, err)
				}
			}
		}
	})

	t.Run("template slugs are unique", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, tmpl := range templates {
			assert.False(t, seen[tmpl.Slug], "duplicate slug: %s", tmpl.Slug)
			seen[tmpl.Slug] = true
		}
	})
}

func TestGetTemplateBySlug(t *testing.T) {
	t.Run("existing slug", func(t *testing.T) {
		templates := GetBuiltinTemplates()
		require.NotEmpty(t, templates)

		tmpl := GetTemplateBySlug(templates[0].Slug)
		require.NotNil(t, tmpl)
		assert.Equal(t, templates[0].Name, tmpl.Name)
	})

	t.Run("nonexistent slug", func(t *testing.T) {
		tmpl := GetTemplateBySlug("nonexistent-slug")
		assert.Nil(t, tmpl)
	})
}

func TestTemplateDefinition_GetPreviewSchemas(t *testing.T) {
	templates := GetBuiltinTemplates()
	require.NotEmpty(t, templates)

	for _, tmpl := range templates {
		previews := tmpl.GetPreviewSchemas()
		assert.Len(t, previews, len(tmpl.Schemas), "preview count should match schema count for %s", tmpl.Slug)

		for i, p := range previews {
			assert.Equal(t, tmpl.Schemas[i].Name, p.Name)
			assert.Equal(t, tmpl.Schemas[i].ApiID, p.ApiID)
		}
	}
}

func TestTemplateDefinition_ToJSON(t *testing.T) {
	templates := GetBuiltinTemplates()
	require.NotEmpty(t, templates)

	for _, tmpl := range templates {
		jsonBytes, err := tmpl.ToJSON()
		require.NoError(t, err, "ToJSON failed for %s", tmpl.Slug)
		assert.NotEmpty(t, jsonBytes)
	}
}
