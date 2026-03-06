package lsp

import (
	"strings"
)

// GetDefinition returns the definition location for a symbol
func GetDefinition(doc *Document, pos Position) *Location {
	word := doc.GetWordAt(pos)
	if word == "" {
		return nil
	}

	schema := doc.GetSchema()
	if schema == nil {
		return nil
	}

	// Look for type definition
	for _, t := range schema.Types {
		if t.Name == word {
			// Find the type definition in the document
			loc := findTypeDefinition(doc, t.Name)
			if loc != nil {
				return loc
			}
		}
	}

	// Look for enum definition
	for _, e := range schema.Enums {
		if e.Name == word {
			loc := findEnumDefinition(doc, e.Name)
			if loc != nil {
				return loc
			}
		}
	}

	return nil
}

// GetReferences returns all references to a symbol
func GetReferences(doc *Document, pos Position, includeDeclaration bool) []Location {
	word := doc.GetWordAt(pos)
	if word == "" {
		return nil
	}

	var locations []Location

	schema := doc.GetSchema()
	if schema == nil {
		return locations
	}

	// Check if it's a type name
	isType := false
	for _, t := range schema.Types {
		if t.Name == word {
			isType = true
			break
		}
	}
	if !isType {
		for _, e := range schema.Enums {
			if e.Name == word {
				isType = true
				break
			}
		}
	}

	if isType {
		// Find all references to this type
		locations = findTypeReferences(doc, word, includeDeclaration)
	}

	return locations
}

func findTypeDefinition(doc *Document, typeName string) *Location {
	for i, line := range doc.Lines {
		// Look for "type TypeName {" pattern
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "type ") {
			rest := strings.TrimPrefix(trimmed, "type ")
			// Get the name (next word)
			parts := strings.Fields(rest)
			if len(parts) > 0 && parts[0] == typeName {
				col := strings.Index(line, typeName)
				return &Location{
					URI: doc.URI,
					Range: Range{
						Start: Position{Line: i, Character: col},
						End:   Position{Line: i, Character: col + len(typeName)},
					},
				}
			}
		}
	}
	return nil
}

func findEnumDefinition(doc *Document, enumName string) *Location {
	for i, line := range doc.Lines {
		// Look for "enum EnumName {" pattern
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "enum ") {
			rest := strings.TrimPrefix(trimmed, "enum ")
			parts := strings.Fields(rest)
			if len(parts) > 0 && parts[0] == enumName {
				col := strings.Index(line, enumName)
				return &Location{
					URI: doc.URI,
					Range: Range{
						Start: Position{Line: i, Character: col},
						End:   Position{Line: i, Character: col + len(enumName)},
					},
				}
			}
		}
	}
	return nil
}

func findTypeReferences(doc *Document, typeName string, includeDeclaration bool) []Location {
	var locations []Location

	for i, line := range doc.Lines {
		// Find all occurrences of the type name
		idx := 0
		for {
			pos := strings.Index(line[idx:], typeName)
			if pos == -1 {
				break
			}
			actualPos := idx + pos

			// Check if it's a whole word
			if isWholeWord(line, actualPos, len(typeName)) {
				isDeclaration := isTypeDeclaration(line, typeName)
				if includeDeclaration || !isDeclaration {
					locations = append(locations, Location{
						URI: doc.URI,
						Range: Range{
							Start: Position{Line: i, Character: actualPos},
							End:   Position{Line: i, Character: actualPos + len(typeName)},
						},
					})
				}
			}

			idx = actualPos + len(typeName)
		}
	}

	return locations
}

func isWholeWord(line string, pos, length int) bool {
	// Check character before
	if pos > 0 {
		c := line[pos-1]
		if isWordChar(c) {
			return false
		}
	}
	// Check character after
	end := pos + length
	if end < len(line) {
		c := line[end]
		if isWordChar(c) {
			return false
		}
	}
	return true
}

func isTypeDeclaration(line, typeName string) bool {
	f := strings.Fields(line)
	return len(f) >= 2 && (f[0] == "type" || f[0] == "enum") && (f[1] == typeName || strings.HasPrefix(f[1], typeName+"{"))
}
