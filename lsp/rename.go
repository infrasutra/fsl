package lsp

import "strings"

// PrepareRenameResult represents the result of a prepare rename request
type PrepareRenameResult struct {
	Range       Range  `json:"range"`
	Placeholder string `json:"placeholder"`
}

// RenameParams represents rename request parameters
type RenameParams struct {
	TextDocumentPositionParams
	NewName string `json:"newName"`
}

// WorkspaceEdit represents changes to many resources managed in the workspace
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

// GetPrepareRename checks if a symbol at the given position can be renamed
func GetPrepareRename(doc *Document, pos Position) *PrepareRenameResult {
	word := doc.GetWordAt(pos)
	if word == "" {
		return nil
	}

	schema := doc.GetSchema()
	if schema == nil {
		return nil
	}

	// Only allow renaming type and enum names
	for _, t := range schema.Types {
		if t.Name == word {
			r := doc.GetWordRange(pos)
			return &PrepareRenameResult{Range: r, Placeholder: word}
		}
	}

	for _, e := range schema.Enums {
		if e.Name == word {
			r := doc.GetWordRange(pos)
			return &PrepareRenameResult{Range: r, Placeholder: word}
		}
	}

	return nil
}

// GetRename computes workspace edits to rename a symbol
func GetRename(doc *Document, pos Position, newName string) *WorkspaceEdit {
	word := doc.GetWordAt(pos)
	if word == "" {
		return nil
	}

	schema := doc.GetSchema()
	if schema == nil {
		return nil
	}

	// Verify it's a renameable symbol
	isRenameable := false
	for _, t := range schema.Types {
		if t.Name == word {
			isRenameable = true
			break
		}
	}
	if !isRenameable {
		for _, e := range schema.Enums {
			if e.Name == word {
				isRenameable = true
				break
			}
		}
	}

	if !isRenameable {
		return nil
	}

	// Find all occurrences in the document and replace them
	edits := findRenameEdits(doc, word, newName)
	if len(edits) == 0 {
		return nil
	}

	return &WorkspaceEdit{
		Changes: map[string][]TextEdit{
			doc.URI: edits,
		},
	}
}

// findRenameEdits finds all occurrences of oldName and creates text edits to replace with newName
func findRenameEdits(doc *Document, oldName, newName string) []TextEdit {
	var edits []TextEdit

	for i, line := range doc.Lines {
		idx := 0
		for {
			pos := strings.Index(line[idx:], oldName)
			if pos == -1 {
				break
			}
			actualPos := idx + pos

			if isWholeWord(line, actualPos, len(oldName)) {
				edits = append(edits, TextEdit{
					Range: Range{
						Start: Position{Line: i, Character: actualPos},
						End:   Position{Line: i, Character: actualPos + len(oldName)},
					},
					NewText: newName,
				})
			}

			idx = actualPos + len(oldName)
		}
	}

	return edits
}
