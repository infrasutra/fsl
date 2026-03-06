package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithDiagnosticsAndExternalTypes(t *testing.T) {
	result := ParseWithDiagnosticsAndExternalTypes(`type Article {
  author: Author! @relation
}`, []string{"Author"})

	require.NotNil(t, result)
	assert.True(t, result.Valid)
	require.NotNil(t, result.Schema)
	require.Len(t, result.Schema.Types, 1)
	assert.Equal(t, "Article", result.Schema.Types[0].Name)
}
