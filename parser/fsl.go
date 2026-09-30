package parser

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
)

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
	return ParseWithDiagnosticsAndOptions(source, Options{})
}

func parseErrorToDiagnostic(errMsg, source string) Diagnostic {
	diag := Diagnostic{
		Severity:    SeverityError,
		Message:     errMsg,
		StartLine:   1,
		StartColumn: 1,
		EndLine:     1,
		EndColumn:   1,
		Source:      "parser",
	}

	var line, col int
	var msg string
	n, _ := fmt.Sscanf(errMsg, "parser error at line %d, column %d: %s", &line, &col, &msg)
	if n >= 2 {
		diag.StartLine = line
		diag.StartColumn = col
		diag.EndLine = line

		diag.EndColumn = col + 1

		if idx := strings.Index(errMsg, ": "); idx != -1 {
			diag.Message = errMsg[idx+2:]
		}
	}

	lines := strings.Split(source, "\n")
	if diag.StartLine > 0 && diag.StartLine <= len(lines) {
		lineContent := lines[diag.StartLine-1]
		if diag.StartColumn <= len(lineContent) {

			endCol := diag.StartColumn
			for endCol <= len(lineContent) && !isWhitespace(lineContent[endCol-1]) {
				endCol++
			}
			diag.EndColumn = endCol
		}
	}

	return diag
}

func validationErrorToDiagnostic(valErr ValidationError, source string) Diagnostic {
	diag := Diagnostic{
		Severity: SeverityError,
		Message:  valErr.Error(),
		Source:   "validator",
	}

	lines := strings.Split(source, "\n")

	if valErr.Field != "" {

		parts := strings.Split(valErr.Field, ".")
		var fieldName string
		if len(parts) >= 2 {
			fieldName = parts[len(parts)-1]
		} else {
			fieldName = valErr.Field
		}

		for i, line := range lines {

			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, fieldName+":") || strings.HasPrefix(trimmed, fieldName+" :") {
				diag.StartLine = i + 1
				diag.StartColumn = strings.Index(line, fieldName) + 1
				diag.EndLine = i + 1
				diag.EndColumn = diag.StartColumn + len(fieldName)
				return diag
			}
		}

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

	if match := regexp.MustCompile(`^(type|enum) '(\w+)' is already defined in the section library$`).FindStringSubmatch(valErr.Message); match != nil {
		declaration := regexp.MustCompile(`^\s*` + match[1] + `\s+(` + match[2] + `)\b`)
		for i, line := range lines {
			if loc := declaration.FindStringSubmatchIndex(line); loc != nil {
				diag.StartLine = i + 1
				diag.StartColumn = loc[2] + 1
				diag.EndLine = i + 1
				diag.EndColumn = loc[3] + 1
				return diag
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

	diag.StartLine = 1
	diag.StartColumn = 1
	diag.EndLine = 1
	diag.EndColumn = 1
	if len(lines) > 0 {
		diag.EndColumn = len(lines[0]) + 1
	}

	return diag
}

func extractQuotedValue(msg string) string {

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

// ValidateData validates document data against compiled schema
func ValidateData(data map[string]any, schema *CompiledSchema) []ValidationError {
	var errors []ValidationError

	fieldMap := make(map[string]*CompiledField)
	for i := range schema.Fields {
		fieldMap[schema.Fields[i].Name] = &schema.Fields[i]
	}

	for _, field := range schema.Fields {
		value, exists := data[field.Name]

		if field.Required && !field.Array && !exists {
			errors = append(errors, ValidationError{
				Field:   field.Name,
				Message: "field is required",
			})
			continue
		}

		if field.Array && field.ArrayReq && !exists {
			errors = append(errors, ValidationError{
				Field:   field.Name,
				Message: "array field is required",
			})
			continue
		}

		if !exists {
			continue
		}

		fieldErrors := validateFieldValue(field.Name, value, &field, schema)
		errors = append(errors, fieldErrors...)
	}

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

func validateFieldValue(fieldName string, value any, field *CompiledField, schema *CompiledSchema) []ValidationError {
	var errors []ValidationError

	if value == nil {
		if field.Array && field.ArrayReq || !field.Array && field.Required {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "value cannot be null for required field",
			})
		}
		return errors
	}

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

		errors = append(errors, validateArrayDecorators(fieldName, arr, field)...)

		for i, elem := range arr {
			elemErrors := validatePrimitiveValue(fmt.Sprintf("%s[%d]", fieldName, i), elem, field.Type, field, schema)
			errors = append(errors, elemErrors...)
		}

		return errors
	}

	errors = append(errors, validatePrimitiveValue(fieldName, value, field.Type, field, schema)...)

	errors = append(errors, validateDecorators(fieldName, value, field)...)

	return errors
}

func validatePrimitiveValue(fieldName string, value any, fieldType string, field *CompiledField, schema *CompiledSchema) []ValidationError {
	var errors []ValidationError

	if values, ok := namedEnumValues(schema, fieldType); ok && len(field.InlineEnum) == 0 {
		str, isString := value.(string)
		if !isString || !slices.Contains(values, str) {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("must be one of: %s", strings.Join(values, ", ")),
			})
		}
		return errors
	}

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

		switch v := value.(type) {
		case float64:
			if v != float64(int64(v)) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "value must be an integer",
				})
			}
		case int64, int, int32:

		default:
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("value must be an integer, got %T", value),
			})
		}

	case TypeFloat:
		switch value.(type) {
		case float64, int64, int, int32, float32:

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

			if _, err := time.Parse("2006-01-02", str); err != nil {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("invalid Date format: %s (expected YYYY-MM-DD)", err),
				})
			}
		}

	case TypeJSON:
		sliceMapping := getSliceMapping(field)
		if len(sliceMapping) > 0 {
			errors = append(errors, validateSliceZone(fieldName, value, sliceMapping, schema)...)
			break
		}

		if !isJSONSerializable(value) {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "value is not JSON serializable",
			})
		}

	case TypeRichText:

		errors = append(errors, validateRichText(fieldName, value, field)...)

	case TypeImage:

		errors = append(errors, validateAssetReference(fieldName, value, TypeImage, field)...)

	case TypeFile:

		errors = append(errors, validateAssetReference(fieldName, value, TypeFile, field)...)

	case "Enum":

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

		if field.IsRelation {
			errors = append(errors, validateRelationReference(fieldName, value)...)
		}

	}

	return errors
}

func getSliceMapping(field *CompiledField) map[string]string {
	mapping := make(map[string]string)

	for _, slice := range field.Slices {
		if slice.Type == "" || slice.Schema == "" {
			continue
		}
		mapping[slice.Type] = slice.Schema
	}

	if len(mapping) > 0 {
		return mapping
	}

	raw, ok := field.Decorators[DecSlices]
	if !ok {
		return mapping
	}

	decoratorMap, ok := raw.(map[string]any)
	if !ok {
		return mapping
	}

	for sliceType, target := range decoratorMap {
		if targetType, ok := target.(string); ok && sliceType != "" && targetType != "" {
			mapping[sliceType] = targetType
		}
	}

	return mapping
}

func buildComponentIndex(schema *CompiledSchema) map[string]*CompiledComponent {
	index := make(map[string]*CompiledComponent, len(schema.Components))
	for i := range schema.Components {
		index[schema.Components[i].Name] = &schema.Components[i]
	}
	return index
}

func validateSliceZone(fieldName string, value any, sliceMapping map[string]string, schema *CompiledSchema) []ValidationError {
	var errors []ValidationError

	items, ok := value.([]any)
	if !ok {
		return []ValidationError{{
			Field:   fieldName,
			Message: "slice zone must be an array of slice objects",
		}}
	}

	componentIndex := buildComponentIndex(schema)

	for idx, item := range items {
		slicePath := fmt.Sprintf("%s[%d]", fieldName, idx)

		sliceObj, ok := item.(map[string]any)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   slicePath,
				Message: "slice item must be an object",
			})
			continue
		}

		sliceType, ok := sliceObj["type"].(string)
		if !ok || strings.TrimSpace(sliceType) == "" {
			errors = append(errors, ValidationError{
				Field:   slicePath,
				Message: "slice item must include a string 'type'",
			})
			continue
		}

		componentName, ok := sliceMapping[sliceType]
		if !ok {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.type", slicePath),
				Message: fmt.Sprintf("unknown slice type '%s'", sliceType),
			})
			continue
		}

		component := componentIndex[componentName]
		if component == nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.type", slicePath),
				Message: fmt.Sprintf("slice type '%s' references missing component '%s'", sliceType, componentName),
			})
			continue
		}

		rawData, exists := sliceObj["data"]
		if !exists {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.data", slicePath),
				Message: "slice item must include a 'data' object",
			})
			continue
		}

		dataMap, ok := rawData.(map[string]any)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.data", slicePath),
				Message: "slice data must be an object",
			})
			continue
		}

		fieldMap := make(map[string]struct{}, len(component.Fields))
		for i := range component.Fields {
			f := component.Fields[i]
			fieldMap[f.Name] = struct{}{}

			value, exists := dataMap[f.Name]
			if f.Required && !f.Array && !exists {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.data.%s", slicePath, f.Name),
					Message: "field is required",
				})
				continue
			}

			if f.Array && f.ArrayReq && !exists {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.data.%s", slicePath, f.Name),
					Message: "array field is required",
				})
				continue
			}

			if !exists {
				continue
			}

			errors = append(errors, validateFieldValue(fmt.Sprintf("%s.data.%s", slicePath, f.Name), value, &f, schema)...)
		}

		for key := range dataMap {
			if _, ok := fieldMap[key]; !ok {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.data.%s", slicePath, key),
					Message: "unexpected field",
				})
			}
		}
	}

	return errors
}

func validateRichText(fieldName string, value any, field *CompiledField) []ValidationError {
	var errors []ValidationError

	blocks, ok := value.([]any)
	if !ok {
		return []ValidationError{{
			Field:   fieldName,
			Message: "RichText must be an array of block objects",
		}}
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

		if _, ok := blockMap["type"].(string); !ok {
			errors = append(errors, ValidationError{
				Field:   blockPath,
				Message: "RichText block must have a 'type' field",
			})
			continue
		}
	}

	return errors
}

func validateAssetReference(fieldName string, value any, assetType string, field *CompiledField) []ValidationError {
	var errors []ValidationError

	assetMap, ok := value.(map[string]any)
	if !ok {
		return []ValidationError{{
			Field:   fieldName,
			Message: fmt.Sprintf("%s must be an asset reference object", assetType),
		}}
	}

	if _, ok := assetMap["url"].(string); !ok {
		errors = append(errors, ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s asset must have a 'url' field", assetType),
		})
	}

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

func validateRelationReference(fieldName string, value any) []ValidationError {

	switch v := value.(type) {
	case string:

		if !isValidUUID(v) {
			return []ValidationError{{
				Field:   fieldName,
				Message: "relation reference must be a valid UUID",
			}}
		}
	case map[string]any:

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

func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return ""
}

func isValidUUID(s string) bool {

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

			if num := toFloat64(value); num != nil {
				if precision, ok := toInt64(decoratorValue); ok {

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

func namedEnumValues(schema *CompiledSchema, name string) ([]string, bool) {
	if schema == nil {
		return nil, false
	}
	for _, enum := range schema.Enums {
		if enum.Name == name {
			return enum.Values, true
		}
	}
	return nil, false
}
