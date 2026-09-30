package cmd

import (
	"testing"

	"github.com/infrasutra/fsl/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilesWithWorkspaceTypes_CrossFileReference(t *testing.T) {
	results, _ := parseFilesWithWorkspaceTypes(map[string]string{
		"article.fsl": `type Article {
  author: Author! @relation
}`,
		"author.fsl": `type Author {
  name: String!
}`,
	})

	article := results["article.fsl"]
	require.NotNil(t, article)
	assert.True(t, article.Valid)
}

func TestParseFilesWithWorkspaceTypes_UnknownTypeStillFails(t *testing.T) {
	results, _ := parseFilesWithWorkspaceTypes(map[string]string{
		"article.fsl": `type Article {
  author: MissingType! @relation
}`,
	})

	article := results["article.fsl"]
	require.NotNil(t, article)
	assert.False(t, article.Valid)
}

func TestParseFilesWithWorkspaceTypes_SectionsFromAnotherFile(t *testing.T) {
	files := map[string]string{
		"sections.fsl": `@description("Opening section")
type Hero {
  heading: String!
}`,
		"landing.fsl": `type Landing {
  sections: JSON! @slices(hero: Hero)
}`,
	}
	results, options := parseFilesWithWorkspaceTypes(files)
	require.True(t, results["landing.fsl"].Valid, results["landing.fsl"].Diagnostics)

	compiled, err := parser.CompileWithOptions(results["landing.fsl"].Schema, "Landing", "landing", false, options["landing.fsl"])
	require.NoError(t, err)
	require.Len(t, compiled.Components, 1)
	assert.True(t, compiled.Components[0].Shared)
	assert.Equal(t, "Opening section", compiled.Components[0].Description)
}

func TestCompileSchemasByType_UsesSectionsFromOtherFiles(t *testing.T) {
	landing := parser.ParseWithDiagnostics("type Landing {\n  sections: JSON! @slices(hero: Hero)\n}").Schema
	sections := parser.ParseWithDiagnostics("enum Tone { calm, bold }\n\ntype Hero {\n  tone: Tone\n}").Schema
	compiled, names, err := compileSchemasByType([]*parser.Schema{landing, sections})
	require.NoError(t, err)
	assert.Equal(t, []string{"Hero", "Landing"}, names)
	assert.True(t, compiled["Landing"].Components[0].Shared)
	assert.Len(t, compiled["Landing"].Enums, 1)
}
