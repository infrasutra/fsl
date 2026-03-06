package lsp

import (
	"encoding/json"

	"github.com/infrasutra/fsl/parser"
)

func (h *Handler) handleFormatting(params json.RawMessage) (interface{}, error) {
	var p DocumentFormattingParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := h.server.GetDocuments().Get(p.TextDocument.URI)
	if doc == nil {
		return []TextEdit{}, nil
	}

	formatted, err := parser.Format(doc.Content)
	if err != nil {
		return []TextEdit{}, nil
	}

	if formatted == doc.Content {
		return []TextEdit{}, nil
	}

	return []TextEdit{{
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: len(doc.Lines) - 1, Character: len(doc.GetLine(len(doc.Lines) - 1))},
		},
		NewText: formatted,
	}}, nil
}
