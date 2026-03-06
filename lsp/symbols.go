package lsp

import (
	"strings"
)

// GetDocumentSymbols returns all symbols in a document
func GetDocumentSymbols(doc *Document) []DocumentSymbol {
	schema := doc.GetSchema()
	if schema == nil {
		return []DocumentSymbol{}
	}

	symbols := make([]DocumentSymbol, 0, len(schema.Types)+len(schema.Enums))

	// Add types
	for _, t := range schema.Types {
		typeRange := findTypeRange(doc, t.Name)
		if typeRange == nil {
			continue
		}

		children := make([]DocumentSymbol, 0, len(t.Fields))
		for _, f := range t.Fields {
			fieldRange := findFieldRange(doc, t.Name, f.Name)
			if fieldRange == nil {
				continue
			}

			detail := formatFieldType(f.Type, f.Array, f.Required)
			if f.IsRelation {
				detail += " → relation"
			}

			fieldSymbol := DocumentSymbol{
				Name:           f.Name,
				Detail:         detail,
				Kind:           SymbolKindField,
				Range:          *fieldRange,
				SelectionRange: *fieldRange,
			}

			children = append(children, fieldSymbol)
		}

		typeSymbol := DocumentSymbol{
			Name:           t.Name,
			Detail:         "type",
			Kind:           SymbolKindClass,
			Range:          *typeRange,
			SelectionRange: *typeRange,
			Children:       children,
		}

		symbols = append(symbols, typeSymbol)
	}

	// Add enums
	for _, e := range schema.Enums {
		enumRange := findEnumRange(doc, e.Name)
		if enumRange == nil {
			continue
		}

		children := make([]DocumentSymbol, 0, len(e.Values))
		for _, v := range e.Values {
			valueRange := findEnumValueRange(doc, e.Name, v)
			if valueRange == nil {
				continue
			}

			children = append(children, DocumentSymbol{
				Name:           v,
				Kind:           SymbolKindEnumMember,
				Range:          *valueRange,
				SelectionRange: *valueRange,
			})
		}

		enumSymbol := DocumentSymbol{
			Name:           e.Name,
			Detail:         "enum",
			Kind:           SymbolKindEnum,
			Range:          *enumRange,
			SelectionRange: *enumRange,
			Children:       children,
		}

		symbols = append(symbols, enumSymbol)
	}

	return symbols
}

func formatFieldType(fieldType string, isArray, isRequired bool) string {
	result := fieldType
	if isArray {
		result = "[" + result + "]"
	}
	if isRequired {
		result += "!"
	}
	return result
}

func findTypeRange(doc *Document, typeName string) *Range {
	inType := false
	startLine := -1
	braceCount := 0

	for i, line := range doc.Lines {
		trimmed := strings.TrimSpace(line)

		if !inType {
			if isTypeDeclaration(line, typeName) ||
				(strings.HasPrefix(trimmed, "@") && containsTypeDeclaration(doc.Lines, i, typeName)) {
				startLine = i
				inType = true
			}
		}

		if inType {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 && startLine != -1 && strings.Contains(line, "}") {
				// Find the type name position
				nameCol := 0
				for j := startLine; j <= i; j++ {
					if idx := strings.Index(doc.Lines[j], typeName); idx != -1 {
						nameCol = idx
						break
					}
				}

				return &Range{
					Start: Position{Line: startLine, Character: nameCol},
					End:   Position{Line: i, Character: len(line)},
				}
			}
		}
	}

	return nil
}

func containsTypeDeclaration(lines []string, fromLine int, typeName string) bool {
	for i := fromLine; i < len(lines) && i < fromLine+5; i++ {
		if isTypeDeclaration(lines[i], typeName) {
			return true
		}
	}
	return false
}

func findEnumRange(doc *Document, enumName string) *Range {
	inEnum := false
	startLine := -1
	braceCount := 0

	for i, line := range doc.Lines {
		if !inEnum {
			if isTypeDeclaration(line, enumName) { // isTypeDeclaration handles both type and enum
				startLine = i
				inEnum = true
			}
		}

		if inEnum {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 && startLine != -1 && strings.Contains(line, "}") {
				nameCol := strings.Index(doc.Lines[startLine], enumName)
				if nameCol == -1 {
					nameCol = 0
				}

				return &Range{
					Start: Position{Line: startLine, Character: nameCol},
					End:   Position{Line: i, Character: len(line)},
				}
			}
		}
	}

	return nil
}

func findFieldRange(doc *Document, typeName, fieldName string) *Range {
	inType := false
	braceCount := 0

	for i, line := range doc.Lines {
		if !inType {
			if isTypeDeclaration(line, typeName) {
				inType = true
			}
		}

		if inType {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Look for field definition
			if isFieldDeclaration(line, fieldName) {
				col := strings.Index(line, fieldName)
				return &Range{
					Start: Position{Line: i, Character: col},
					End:   Position{Line: i, Character: len(strings.TrimRight(line, " \t"))},
				}
			}

			if braceCount <= 0 {
				break
			}
		}
	}

	return nil
}

func isFieldDeclaration(line, fieldName string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, ":") {
		return false
	}
	parts := strings.SplitN(trimmed, ":", 2)
	return strings.TrimSpace(parts[0]) == fieldName
}

func findEnumValueRange(doc *Document, enumName, value string) *Range {
	inEnum := false
	braceCount := 0

	for i, line := range doc.Lines {
		if !inEnum {
			if isTypeDeclaration(line, enumName) {
				inEnum = true
			}
		}

		if inEnum {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Look for enum value
			trimmed := strings.TrimSpace(line)
			if trimmed == value {
				col := strings.Index(line, value)
				return &Range{
					Start: Position{Line: i, Character: col},
					End:   Position{Line: i, Character: col + len(value)},
				}
			}

			if braceCount <= 0 {
				break
			}
		}
	}

	return nil
}
