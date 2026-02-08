package parser

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// =============================================================================
// Diagnostic Types for IDE Integration
// =============================================================================

// DiagnosticSeverity represents the severity level of a diagnostic
type DiagnosticSeverity int

const (
	SeverityError   DiagnosticSeverity = 1
	SeverityWarning DiagnosticSeverity = 2
	SeverityInfo    DiagnosticSeverity = 3
	SeverityHint    DiagnosticSeverity = 4
)

// Diagnostic represents a single error/warning with position information
type Diagnostic struct {
	Severity    DiagnosticSeverity `json:"severity"`
	Message     string             `json:"message"`
	StartLine   int                `json:"startLine"`   // 1-indexed
	StartColumn int                `json:"startColumn"` // 1-indexed
	EndLine     int                `json:"endLine"`     // 1-indexed
	EndColumn   int                `json:"endColumn"`   // 1-indexed
	Source      string             `json:"source"`      // "parser" or "validator"
}

// DiagnosticsResult contains the result of parsing with diagnostics
type DiagnosticsResult struct {
	Valid       bool         `json:"valid"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Schema      *Schema      `json:"-"` // Parsed schema (if successful)
}

// ParseWithDiagnostics parses FSL and returns structured diagnostics for IDE integration
func ParseWithDiagnostics(source string) *DiagnosticsResult {
	result := &DiagnosticsResult{
		Valid:       true,
		Diagnostics: []Diagnostic{},
	}

	// Parse the source
	lexer := NewLexer(source)
	parser := NewParser(lexer)
	schema, err := parser.ParseSchema()

	if err != nil {
		result.Valid = false
		// Extract line/column from parser error
		diag := parseErrorToDiagnostic(err.Error(), source)
		result.Diagnostics = append(result.Diagnostics, diag)
		return result
	}

	result.Schema = schema

	// Validate the schema
	validator := NewValidator(schema)
	validationErrors := validator.Validate()

	if len(validationErrors) > 0 {
		result.Valid = false
		for _, valErr := range validationErrors {
			diag := validationErrorToDiagnostic(valErr, source)
			result.Diagnostics = append(result.Diagnostics, diag)
		}
	}

	return result
}

// parseErrorToDiagnostic converts a parser error string to a Diagnostic
func parseErrorToDiagnostic(errMsg string, source string) Diagnostic {
	diag := Diagnostic{
		Severity:    SeverityError,
		Message:     errMsg,
		StartLine:   1,
		StartColumn: 1,
		EndLine:     1,
		EndColumn:   1,
		Source:      "parser",
	}

	// Try to extract line and column from error message
	// Format: "parser error at line X, column Y: message"
	var line, col int
	var msg string
	n, _ := fmt.Sscanf(errMsg, "parser error at line %d, column %d: %s", &line, &col, &msg)
	if n >= 2 {
		diag.StartLine = line
		diag.StartColumn = col
		diag.EndLine = line
		// Find end of token/word for highlighting
		diag.EndColumn = col + 1
		// Extract the actual message after the position info
		if idx := strings.Index(errMsg, ": "); idx != -1 {
			diag.Message = errMsg[idx+2:]
		}
	}

	// Extend end column to cover more context
	lines := strings.Split(source, "\n")
	if diag.StartLine > 0 && diag.StartLine <= len(lines) {
		lineContent := lines[diag.StartLine-1]
		if diag.StartColumn <= len(lineContent) {
			// Find end of word/token
			endCol := diag.StartColumn
			for endCol <= len(lineContent) && !isWhitespace(lineContent[endCol-1]) {
				endCol++
			}
			diag.EndColumn = endCol
		}
	}

	return diag
}

// validationErrorToDiagnostic converts a ValidationError to a Diagnostic with position
func validationErrorToDiagnostic(valErr ValidationError, source string) Diagnostic {
	diag := Diagnostic{
		Severity: SeverityError,
		Message:  valErr.Error(),
		Source:   "validator",
	}

	// Try to find the field in the source
	lines := strings.Split(source, "\n")

	if valErr.Field != "" {
		// Field path like "TypeName.fieldName"
		parts := strings.Split(valErr.Field, ".")
		var fieldName string
		if len(parts) >= 2 {
			fieldName = parts[len(parts)-1]
		} else {
			fieldName = valErr.Field
		}

		// Search for the field name in source
		for i, line := range lines {
			// Look for field definition pattern: fieldName:
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, fieldName+":") || strings.HasPrefix(trimmed, fieldName+" :") {
				diag.StartLine = i + 1
				diag.StartColumn = strings.Index(line, fieldName) + 1
				diag.EndLine = i + 1
				diag.EndColumn = diag.StartColumn + len(fieldName)
				return diag
			}
		}

		// If field contains decorator error, search for decorator
		if strings.Contains(valErr.Message, "@") {
			decoratorMatch := regexp.MustCompile(`@(\w+)`).FindStringSubmatch(valErr.Message)
			if len(decoratorMatch) >= 2 {
				decoratorName := decoratorMatch[1]
				for i, line := range lines {
					if strings.Contains(line, "@"+decoratorName) {
						diag.StartLine = i + 1
						diag.StartColumn = strings.Index(line, "@"+decoratorName) + 1
						diag.EndLine = i + 1
						diag.EndColumn = diag.StartColumn + len(decoratorName) + 1
						return diag
					}
				}
			}
		}
	}

	// If we couldn't find specific location, try to match error message patterns
	if strings.Contains(valErr.Message, "duplicate type name") {
		typeName := extractQuotedValue(valErr.Message)
		if typeName != "" {
			for i, line := range lines {
				if strings.Contains(line, "type "+typeName) {
					diag.StartLine = i + 1
					diag.StartColumn = strings.Index(line, typeName) + 1
					diag.EndLine = i + 1
					diag.EndColumn = diag.StartColumn + len(typeName)
					return diag
				}
			}
		}
	}

	if strings.Contains(valErr.Message, "unknown type") {
		typeName := extractQuotedValue(valErr.Message)
		if typeName != "" {
			for i, line := range lines {
				if strings.Contains(line, ": "+typeName) || strings.Contains(line, ":"+typeName) {
					idx := strings.Index(line, typeName)
					if idx != -1 {
						diag.StartLine = i + 1
						diag.StartColumn = idx + 1
						diag.EndLine = i + 1
						diag.EndColumn = diag.StartColumn + len(typeName)
						return diag
					}
				}
			}
		}
	}

	// Default: first line of schema
	diag.StartLine = 1
	diag.StartColumn = 1
	diag.EndLine = 1
	diag.EndColumn = 1
	if len(lines) > 0 {
		diag.EndColumn = len(lines[0]) + 1
	}

	return diag
}

// extractQuotedValue extracts a value from error messages like "unknown type: Foo"
func extractQuotedValue(msg string) string {
	// Try ": value" pattern
	if idx := strings.LastIndex(msg, ": "); idx != -1 {
		return strings.TrimSpace(msg[idx+2:])
	}
	return ""
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Parse parses FSL source code and returns the AST
func Parse(source string) (*Schema, error) {
	lexer := NewLexer(source)
	parser := NewParser(lexer)
	schema, err := parser.ParseSchema()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Validate the schema
	if err := ValidateSchema(schema); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return schema, nil
}

// ParseAndCompile parses and compiles FSL to CompiledSchema
func ParseAndCompile(source, name, apiID string, singleton bool) (*CompiledSchema, error) {
	schema, err := Parse(source)
	if err != nil {
		return nil, err
	}

	compiled, err := Compile(schema, name, apiID, singleton)
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}

	return compiled, nil
}

// ParseAndCompileWithExternalTypes parses and compiles FSL while treating specified types as valid relation targets.
// This is useful for templates where multiple schemas can reference each other's types.
func ParseAndCompileWithExternalTypes(source, name, apiID string, singleton bool, externalTypes []string) (*CompiledSchema, error) {
	// Parse without validation first
	lexer := NewLexer(source)
	parser := NewParser(lexer)
	schema, err := parser.ParseSchema()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Validate with external types
	if err := ValidateSchemaWithExternalTypes(schema, externalTypes); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	compiled, err := Compile(schema, name, apiID, singleton)
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}

	return compiled, nil
}

// ValidateData validates document data against compiled schema
func ValidateData(data map[string]any, schema *CompiledSchema) []ValidationError {
	var errors []ValidationError

	// Create a map of field definitions for quick lookup
	fieldMap := make(map[string]*CompiledField)
	for i := range schema.Fields {
		fieldMap[schema.Fields[i].Name] = &schema.Fields[i]
	}

	// Check required fields
	for _, field := range schema.Fields {
		value, exists := data[field.Name]

		// Check if field is required
		if field.Required && !exists {
			errors = append(errors, ValidationError{
				Field:   field.Name,
				Message: "field is required",
			})
			continue
		}

		// Check if array field is required
		if field.Array && field.ArrayReq && !exists {
			errors = append(errors, ValidationError{
				Field:   field.Name,
				Message: "array field is required",
			})
			continue
		}

		// If field doesn't exist and is not required, skip validation
		if !exists {
			continue
		}

		// Validate field value
		fieldErrors := validateFieldValue(field.Name, value, &field)
		errors = append(errors, fieldErrors...)
	}

	// Check for unexpected fields
	for fieldName := range data {
		if _, exists := fieldMap[fieldName]; !exists {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "unexpected field",
			})
		}
	}

	return errors
}

func validateFieldValue(fieldName string, value any, field *CompiledField) []ValidationError {
	var errors []ValidationError

	// Handle nil values
	if value == nil {
		if field.Required {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "value cannot be null for required field",
			})
		}
		return errors
	}

	// Handle array types
	if field.Array {
		arr, ok := value.([]any)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "value must be an array",
			})
			return errors
		}

		if field.ArrayReq && len(arr) == 0 {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "array cannot be empty",
			})
		}

		// Validate array-level decorators (minItems, maxItems)
		errors = append(errors, validateArrayDecorators(fieldName, arr, field)...)

		// Validate each array element
		for i, elem := range arr {
			elemErrors := validatePrimitiveValue(fmt.Sprintf("%s[%d]", fieldName, i), elem, field.Type, field)
			errors = append(errors, elemErrors...)
		}

		return errors
	}

	// Validate primitive value
	errors = append(errors, validatePrimitiveValue(fieldName, value, field.Type, field)...)

	// Validate decorators
	errors = append(errors, validateDecorators(fieldName, value, field)...)

	return errors
}

func validatePrimitiveValue(fieldName string, value any, fieldType string, field *CompiledField) []ValidationError {
	var errors []ValidationError

	switch fieldType {
	case TypeString, TypeText:
		str, ok := value.(string)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("value must be a string, got %T", value),
			})
		} else if fieldType == TypeString && strings.Contains(str, "\n") {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "String type cannot contain newlines (use Text instead)",
			})
		}

	case TypeInt:
		// Accept both float64 (from JSON) and int64
		switch v := value.(type) {
		case float64:
			if v != float64(int64(v)) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "value must be an integer",
				})
			}
		case int64, int, int32:
			// Valid integer types
		default:
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("value must be an integer, got %T", value),
			})
		}

	case TypeFloat:
		switch value.(type) {
		case float64, int64, int, int32, float32:
			// Valid numeric types
		default:
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("value must be a number, got %T", value),
			})
		}

	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("value must be a boolean, got %T", value),
			})
		}

	case TypeDateTime:
		str, ok := value.(string)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "DateTime must be a string in ISO 8601 format",
			})
		} else {
			// Try to parse as RFC3339 (ISO 8601)
			if _, err := time.Parse(time.RFC3339, str); err != nil {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("invalid DateTime format: %s (expected ISO 8601)", err),
				})
			}
		}

	case TypeDate:
		str, ok := value.(string)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "Date must be a string in YYYY-MM-DD format",
			})
		} else {
			// Parse as date only
			if _, err := time.Parse("2006-01-02", str); err != nil {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("invalid Date format: %s (expected YYYY-MM-DD)", err),
				})
			}
		}

	case TypeJSON:
		// JSON can be any type - no validation needed
		// But we should ensure it's serializable
		if !isJSONSerializable(value) {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "value is not JSON serializable",
			})
		}

	case TypeRichText:
		// RichText must be an array of block objects
		errors = append(errors, validateRichText(fieldName, value, field)...)

	case TypeImage:
		// Image must be an asset reference object
		errors = append(errors, validateAssetReference(fieldName, value, TypeImage, field)...)

	case TypeFile:
		// File must be an asset reference object
		errors = append(errors, validateAssetReference(fieldName, value, TypeFile, field)...)

	case "Enum":
		// Inline enum validation
		if len(field.InlineEnum) > 0 {
			str, ok := value.(string)
			if !ok {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("enum value must be a string, got %T", value),
				})
			} else {
				valid := false
				for _, v := range field.InlineEnum {
					if v == str {
						valid = true
						break
					}
				}
				if !valid {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("invalid enum value: %s (allowed: %v)", str, field.InlineEnum),
					})
				}
			}
		}

	default:
		// Could be a named enum, custom type reference, or relation
		// For relations, validate as reference object
		if field.IsRelation {
			errors = append(errors, validateRelationReference(fieldName, value)...)
		}
		// Named enums and custom types need schema context for full validation
	}

	return errors
}

// validateRichText validates RichText content blocks
func validateRichText(fieldName string, value any, field *CompiledField) []ValidationError {
	var errors []ValidationError

	blocks, ok := value.([]any)
	if !ok {
		return []ValidationError{{
			Field:   fieldName,
			Message: "RichText must be an array of block objects",
		}}
	}

	// Get allowed blocks from decorator
	var allowedBlocks map[string]bool
	if blocksVal, ok := field.Decorators[DecBlocks]; ok {
		allowedBlocks = make(map[string]bool)
		switch v := blocksVal.(type) {
		case string:
			allowedBlocks[v] = true
		case []any:
			for _, b := range v {
				if str, ok := b.(string); ok {
					allowedBlocks[str] = true
				}
			}
		}
	}

	for i, block := range blocks {
		blockPath := fmt.Sprintf("%s[%d]", fieldName, i)
		blockMap, ok := block.(map[string]any)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   blockPath,
				Message: "RichText block must be an object",
			})
			continue
		}

		// Check block has a type
		blockType, ok := blockMap["type"].(string)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   blockPath,
				Message: "RichText block must have a 'type' field",
			})
			continue
		}

		// Validate block type if restrictions are specified
		if allowedBlocks != nil && !allowedBlocks[blockType] {
			errors = append(errors, ValidationError{
				Field:   blockPath,
				Message: fmt.Sprintf("RichText block type '%s' is not allowed", blockType),
			})
		}
	}

	return errors
}

// validateAssetReference validates Image/File asset references
func validateAssetReference(fieldName string, value any, assetType string, field *CompiledField) []ValidationError {
	var errors []ValidationError

	assetMap, ok := value.(map[string]any)
	if !ok {
		return []ValidationError{{
			Field:   fieldName,
			Message: fmt.Sprintf("%s must be an asset reference object", assetType),
		}}
	}

	// Check required fields
	if _, ok := assetMap["url"].(string); !ok {
		errors = append(errors, ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s asset must have a 'url' field", assetType),
		})
	}

	// Validate format if @formats decorator is present
	if formatsVal, ok := field.Decorators[DecFormats]; ok {
		if filename, ok := assetMap["filename"].(string); ok {
			ext := getFileExtension(filename)
			if ext != "" {
				allowed := false
				switch v := formatsVal.(type) {
				case string:
					allowed = strings.EqualFold(ext, v)
				case []any:
					for _, f := range v {
						if str, ok := f.(string); ok && strings.EqualFold(ext, str) {
							allowed = true
							break
						}
					}
				}
				if !allowed {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("file format '%s' is not allowed", ext),
					})
				}
			}
		}
	}

	// Validate size if @maxSize decorator is present
	if maxSizeVal, ok := field.Decorators[DecMaxSize]; ok {
		if size, ok := assetMap["size"]; ok {
			if maxSize, ok := toInt64(maxSizeVal); ok {
				if sizeInt, ok := toInt64(size); ok {
					if sizeInt > maxSize {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("file size %d exceeds maximum of %d bytes", sizeInt, maxSize),
						})
					}
				}
			}
		}
	}

	return errors
}

// validateRelationReference validates a relation reference
func validateRelationReference(fieldName string, value any) []ValidationError {
	// Relation references can be:
	// 1. A UUID string
	// 2. An object with an "id" field
	switch v := value.(type) {
	case string:
		// Should be a valid UUID
		if !isValidUUID(v) {
			return []ValidationError{{
				Field:   fieldName,
				Message: "relation reference must be a valid UUID",
			}}
		}
	case map[string]any:
		// Should have an "id" field
		if id, ok := v["id"].(string); ok {
			if !isValidUUID(id) {
				return []ValidationError{{
					Field:   fieldName,
					Message: "relation reference 'id' must be a valid UUID",
				}}
			}
		} else {
			return []ValidationError{{
				Field:   fieldName,
				Message: "relation reference must have an 'id' field",
			}}
		}
	default:
		return []ValidationError{{
			Field:   fieldName,
			Message: fmt.Sprintf("relation reference must be a UUID string or object, got %T", value),
		}}
	}
	return nil
}

// getFileExtension extracts the file extension from a filename
func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return ""
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(s string) bool {
	// Simple UUID format check (8-4-4-4-12 hex digits)
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func validateDecorators(fieldName string, value any, field *CompiledField) []ValidationError {
	var errors []ValidationError

	for decoratorName, decoratorValue := range field.Decorators {
		switch decoratorName {
		case DecMaxLength:
			if str, ok := value.(string); ok {
				if maxLen, ok := toInt64(decoratorValue); ok {
					if int64(len(str)) > maxLen {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("length exceeds maximum of %d", maxLen),
						})
					}
				}
			}

		case DecMinLength:
			if str, ok := value.(string); ok {
				if minLen, ok := toInt64(decoratorValue); ok {
					if int64(len(str)) < minLen {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("length is less than minimum of %d", minLen),
						})
					}
				}
			}

		case DecMin:
			if num := toFloat64(value); num != nil {
				if minVal := toFloat64(decoratorValue); minVal != nil {
					if *num < *minVal {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("value is less than minimum of %v", minVal),
						})
					}
				}
			}

		case DecMax:
			if num := toFloat64(value); num != nil {
				if maxVal := toFloat64(decoratorValue); maxVal != nil {
					if *num > *maxVal {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("value exceeds maximum of %v", maxVal),
						})
					}
				}
			}

		case DecPattern:
			if str, ok := value.(string); ok {
				if pattern, ok := decoratorValue.(string); ok {
					matched, err := regexp.MatchString(pattern, str)
					if err != nil {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("invalid pattern: %s", err),
						})
					} else if !matched {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("value does not match pattern: %s", pattern),
						})
					}
				}
			}

		case DecPrecision:
			// Validate float precision
			if num := toFloat64(value); num != nil {
				if precision, ok := toInt64(decoratorValue); ok {
					// Check if value has more decimal places than allowed
					formatted := fmt.Sprintf("%.*f", precision, *num)
					var reparsed float64
					fmt.Sscanf(formatted, "%f", &reparsed)
					if *num != reparsed {
						errors = append(errors, ValidationError{
							Field:   fieldName,
							Message: fmt.Sprintf("value exceeds precision of %d decimal places", precision),
						})
					}
				}
			}
		}
	}

	return errors
}

// validateArrayDecorators validates array-specific decorators (minItems, maxItems)
func validateArrayDecorators(fieldName string, arr []any, field *CompiledField) []ValidationError {
	var errors []ValidationError

	if minItems, ok := field.Decorators[DecMinItems]; ok {
		if min, ok := toInt64(minItems); ok {
			if int64(len(arr)) < min {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("array has %d items, minimum is %d", len(arr), min),
				})
			}
		}
	}

	if maxItems, ok := field.Decorators[DecMaxItems]; ok {
		if max, ok := toInt64(maxItems); ok {
			if int64(len(arr)) > max {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("array has %d items, maximum is %d", len(arr), max),
				})
			}
		}
	}

	return errors
}

// Helper functions

func toInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func toFloat64(value any) *float64 {
	var result float64
	switch v := value.(type) {
	case float64:
		result = v
	case float32:
		result = float64(v)
	case int64:
		result = float64(v)
	case int:
		result = float64(v)
	case int32:
		result = float64(v)
	default:
		return nil
	}
	return &result
}

func isJSONSerializable(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if !isJSONSerializable(v.Index(i).Interface()) {
				return false
			}
		}
		return true
	case reflect.Map:
		// Check if keys are strings and values are serializable
		for _, key := range v.MapKeys() {
			if key.Kind() != reflect.String {
				return false
			}
			if !isJSONSerializable(v.MapIndex(key).Interface()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
