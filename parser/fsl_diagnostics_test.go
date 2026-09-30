package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithDiagnosticsAndOptions(t *testing.T) {
	result := ParseWithDiagnosticsAndOptions(`type Article {
  author: Author! @relation
}`, Options{ExternalTypes: []string{"Author"}})

	require.NotNil(t, result)
	assert.True(t, result.Valid)
	require.NotNil(t, result.Schema)
	require.Len(t, result.Schema.Types, 1)
	assert.Equal(t, "Article", result.Schema.Types[0].Name)
}
