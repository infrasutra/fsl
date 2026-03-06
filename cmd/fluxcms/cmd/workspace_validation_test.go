package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilesWithWorkspaceTypes_CrossFileReference(t *testing.T) {
	results := parseFilesWithWorkspaceTypes(map[string]string{
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
	results := parseFilesWithWorkspaceTypes(map[string]string{
		"article.fsl": `type Article {
  author: MissingType! @relation
}`,
	})

	article := results["article.fsl"]
	require.NotNil(t, article)
	assert.False(t, article.Valid)
}
