package parser

import (
	"fmt"
	"regexp"
	"strings"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

type Validator struct {
	schema        *Schema
	errors        []ValidationError
	typeNames     map[string]bool
	enumNames     map[string]bool            // Named enum definitions
	enumValues    map[string]map[string]bool // enumName -> value -> exists
	fieldNames    map[string]map[string]bool // typeName -> fieldName -> exists
	relations     map[string][]string        // typeName -> list of relation field names
	externalTypes map[string]bool            // External types that should be treated as valid (for templates)
}

func NewValidator(schema *Schema) *Validator {
	return &Validator{
		schema:        schema,
		errors:        []ValidationError{},
		typeNames:     make(map[string]bool),
		enumNames:     make(map[string]bool),
		enumValues:    make(map[string]map[string]bool),
		fieldNames:    make(map[string]map[string]bool),
		relations:     make(map[string][]string),
		externalTypes: make(map[string]bool),
	}
}

// NewValidatorWithExternalTypes creates a validator that knows about external types
// (types defined in other schemas that should be treated as valid relation targets)
func NewValidatorWithExternalTypes(schema *Schema, externalTypes []string) *Validator {
	v := NewValidator(schema)
	for _, t := range externalTypes {
		v.externalTypes[t] = true
	}
	return v
}

func (v *Validator) Validate() []ValidationError {
	// First pass: collect enum names
	for _, enumDef := range v.schema.Enums {
		if v.enumNames[enumDef.Name] {
			v.addError("", fmt.Sprintf("duplicate enum name: %s", enumDef.Name))
		}
		v.enumNames[enumDef.Name] = true
		v.enumValues[enumDef.Name] = make(map[string]bool)
		for _, val := range enumDef.Values {
			v.enumValues[enumDef.Name][val] = true
		}
	}

	// Second pass: collect type names (check conflict with enum names)
	for _, typeDef := range v.schema.Types {
		if v.typeNames[typeDef.Name] {
			v.addError("", fmt.Sprintf("duplicate type name: %s", typeDef.Name))
		}
		if v.enumNames[typeDef.Name] {
			v.addError("", fmt.Sprintf("type name conflicts with enum name: %s", typeDef.Name))
		}
		v.typeNames[typeDef.Name] = true
		v.fieldNames[typeDef.Name] = make(map[string]bool)
	}

	// Third pass: validate enums
	for _, enumDef := range v.schema.Enums {
		v.validateEnumDef(&enumDef)
	}

	// Fourth pass: validate each type and collect relations
	for i := range v.schema.Types {
		v.validateTypeDef(&v.schema.Types[i])
	}

	// Fifth pass: validate relations (cross-type references)
	v.validateRelations()

	return v.errors
}

func (v *Validator) validateEnumDef(enumDef *EnumDef) {
	if enumDef.Name == "" {
		v.addError("", "enum name cannot be empty")
		return
	}

	if len(enumDef.Values) == 0 {
		v.addError(enumDef.Name, "enum must have at least one value")
		return
	}

	// Check for duplicate values
	seen := make(map[string]bool)
	for _, val := range enumDef.Values {
		if seen[val] {
			v.addError(enumDef.Name, fmt.Sprintf("duplicate enum value: %s", val))
		}
		seen[val] = true
	}
}

func (v *Validator) validateTypeDef(typeDef *TypeDef) {
	// Validate type name
	if typeDef.Name == "" {
		v.addError("", "type name cannot be empty")
		return
	}

	// Validate fields
	if len(typeDef.Fields) == 0 {
		v.addError(typeDef.Name, "type must have at least one field")
	}

	for i := range typeDef.Fields {
		v.validateField(typeDef.Name, &typeDef.Fields[i])
	}
}

func (v *Validator) validateField(typeName string, field *FieldDef) {
	fieldPath := fmt.Sprintf("%s.%s", typeName, field.Name)

	// Check for empty field name
	if field.Name == "" {
		v.addError(fieldPath, "field name cannot be empty")
		return
	}

	// Check for reserved field names
	if ReservedFieldNames[field.Name] {
		v.addError(fieldPath, fmt.Sprintf("field name '%s' is reserved", field.Name))
	}

	// Check for duplicate field names
	if v.fieldNames[typeName][field.Name] {
		v.addError(fieldPath, fmt.Sprintf("duplicate field name: %s", field.Name))
	}
	v.fieldNames[typeName][field.Name] = true

	// Validate field type
	if field.Type == "" {
		v.addError(fieldPath, "field type cannot be empty")
		return
	}

	// Check if type exists (builtin, enum, defined type, or external type)
	isBuiltin := BuiltinTypes[field.Type]
	isEnum := v.enumNames[field.Type]
	isDefinedType := v.typeNames[field.Type]
	isExternalType := v.externalTypes[field.Type]
	isInlineEnum := field.Type == "Enum" && len(field.InlineEnum) > 0

	if !isBuiltin && !isEnum && !isDefinedType && !isExternalType && !isInlineEnum {
		v.addError(fieldPath, fmt.Sprintf("unknown type: %s", field.Type))
	}

	// Auto-detect relations: if field type is a defined type or external type (not builtin, not enum, not inline enum),
	// it's a relation. The @relation decorator is still supported for advanced features like inverse/onDelete.
	if (isDefinedType || isExternalType) && !isBuiltin && !isEnum && !isInlineEnum {
		field.IsRelation = true
	}

	// Track relation fields for cross-type validation
	if field.IsRelation {
		v.relations[typeName] = append(v.relations[typeName], field.Name)
	}

	// Validate inline enum values
	if isInlineEnum {
		seen := make(map[string]bool)
		for _, val := range field.InlineEnum {
			if seen[val] {
				v.addError(fieldPath, fmt.Sprintf("duplicate inline enum value: %s", val))
			}
			seen[val] = true
		}
	}

	// Validate decorators
	for decoratorName, decoratorValue := range field.Decorators {
		v.validateDecorator(fieldPath, field.Type, field, decoratorName, decoratorValue)
	}
}

func (v *Validator) validateDecorator(fieldPath, fieldType string, field *FieldDef, decoratorName string, decoratorValue any) {
	switch decoratorName {
	case DecMaxLength, DecMinLength:
		// Only valid for String and Text
		if fieldType != TypeString && fieldType != TypeText {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with String or Text types", decoratorName))
		}
		// Validate value is a positive integer
		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, fmt.Sprintf("@%s value must be a positive integer", decoratorName))
		}

	case DecMin, DecMax:
		// Only valid for Int and Float
		if fieldType != TypeInt && fieldType != TypeFloat {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with Int or Float types", decoratorName))
		}
		// Validate value is a number
		if !v.isNumber(decoratorValue) {
			v.addError(fieldPath, fmt.Sprintf("@%s value must be a number", decoratorName))
		}

	case DecPattern:
		// Only valid for String
		if fieldType != TypeString {
			v.addError(fieldPath, "@pattern can only be used with String type")
		}
		// Validate value is a string
		if _, ok := decoratorValue.(string); !ok {
			v.addError(fieldPath, "@pattern value must be a string")
		}

	case DecDefault:
		// Validate default value type matches field type
		if !v.validateDefaultValue(fieldType, decoratorValue) {
			v.addError(fieldPath, fmt.Sprintf("@default value type does not match field type %s", fieldType))
		}

	case DecUnique, DecIndex, DecSearchable, DecHidden:
		// These are boolean flags, no additional validation needed
		// Value should be true or omitted

	case DecRelation:
		// Validate relation decorator
		v.validateRelationDecorator(fieldPath, field, decoratorValue)

	case DecSlices:
		// Only valid for JSON and only on non-array fields
		if fieldType != TypeJSON {
			v.addError(fieldPath, "@slices can only be used with JSON type")
		}
		if field.Array {
			v.addError(fieldPath, "@slices cannot be used on array fields")
		}
		v.validateSlicesDecorator(fieldPath, decoratorValue)

	case DecMaxSize:
		// Only valid for Image and File
		if fieldType != TypeImage && fieldType != TypeFile {
			v.addError(fieldPath, "@maxSize can only be used with Image or File types")
		}
		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, "@maxSize value must be a positive integer (bytes)")
		}

	case DecFormats:
		// Only valid for Image and File
		if fieldType != TypeImage && fieldType != TypeFile {
			v.addError(fieldPath, "@formats can only be used with Image or File types")
		}
		v.validateFormatsDecorator(fieldPath, fieldType, decoratorValue)

	case DecPrecision:
		// Only valid for Float
		if fieldType != TypeFloat {
			v.addError(fieldPath, "@precision can only be used with Float type")
		}
		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, "@precision value must be a positive integer")
		}

	case DecMinItems:
		// Only valid for array fields
		if !field.Array {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with array types", decoratorName))
		}
		if !v.isNonNegativeInt(decoratorValue) {
			v.addError(fieldPath, "@minItems value must be a non-negative integer")
		}

	case DecMaxItems:
		// Only valid for array fields
		if !field.Array {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with array types", decoratorName))
		}
		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, "@maxItems value must be a positive integer")
		}

	default:
		// Unknown decorator - could warn but not error
		v.addError(fieldPath, fmt.Sprintf("unknown decorator: @%s", decoratorName))
	}
}

func (v *Validator) validateRelationDecorator(fieldPath string, field *FieldDef, decoratorValue any) {
	// @relation can be used with or without arguments
	// @relation - basic relation
	// @relation(inverse: "fieldName") - bidirectional relation

	if decoratorValue == true {
		// Basic relation with no arguments - valid
		return
	}

	// Check for named arguments
	if args, ok := decoratorValue.(map[string]any); ok {
		if inverse, ok := args["inverse"]; ok {
			if _, isStr := inverse.(string); !isStr {
				v.addError(fieldPath, "@relation inverse argument must be a string")
			}
		}
		if onDelete, ok := args["onDelete"]; ok {
			if str, isStr := onDelete.(string); isStr {
				validOnDelete := map[string]bool{"cascade": true, "restrict": true, "setNull": true}
				if !validOnDelete[str] {
					v.addError(fieldPath, "@relation onDelete must be 'cascade', 'restrict', or 'setNull'")
				}
			} else {
				v.addError(fieldPath, "@relation onDelete argument must be a string")
			}
		}
	}
}

var sliceTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func (v *Validator) validateSlicesDecorator(fieldPath string, decoratorValue any) {
	sliceMap, ok := decoratorValue.(map[string]any)
	if !ok || len(sliceMap) == 0 {
		v.addError(fieldPath, "@slices must define named slice mappings, e.g. @slices(hero: HeroSlice, faq: FaqSlice)")
		return
	}

	for sliceType, targetAny := range sliceMap {
		if !sliceTypePattern.MatchString(sliceType) {
			v.addError(fieldPath, fmt.Sprintf("invalid slice type '%s': must be snake_case and start with a lowercase letter", sliceType))
		}

		targetType, ok := targetAny.(string)
		if !ok || strings.TrimSpace(targetType) == "" {
			v.addError(fieldPath, fmt.Sprintf("slice '%s' must reference a type name", sliceType))
			continue
		}

		if BuiltinTypes[targetType] {
			v.addError(fieldPath, fmt.Sprintf("slice '%s' cannot reference builtin type '%s'", sliceType, targetType))
			continue
		}

		if v.enumNames[targetType] {
			v.addError(fieldPath, fmt.Sprintf("slice '%s' cannot reference enum '%s'", sliceType, targetType))
			continue
		}

		if !v.typeNames[targetType] {
			v.addError(fieldPath, fmt.Sprintf("slice '%s' references unknown type '%s'", sliceType, targetType))
		}
	}
}

func (v *Validator) validateFormatsDecorator(fieldPath, fieldType string, decoratorValue any) {
	validFormats := ValidFileFormats
	if fieldType == TypeImage {
		validFormats = ValidImageFormats
	}

	switch val := decoratorValue.(type) {
	case string:
		if !validFormats[val] {
			v.addError(fieldPath, fmt.Sprintf("unknown format: %s", val))
		}
	case []any:
		for _, item := range val {
			if str, ok := item.(string); ok {
				if !validFormats[str] {
					v.addError(fieldPath, fmt.Sprintf("unknown format: %s", str))
				}
			} else {
				v.addError(fieldPath, "@formats values must be strings")
			}
		}
	default:
		v.addError(fieldPath, "@formats must be a string or array of strings")
	}
}

// validateRelations performs cross-type validation for relations
func (v *Validator) validateRelations() {
	for _, typeDef := range v.schema.Types {
		for _, field := range typeDef.Fields {
			if !field.IsRelation {
				continue
			}

			fieldPath := fmt.Sprintf("%s.%s", typeDef.Name, field.Name)
			targetType := field.Type

			// Check if target type exists (must be a defined type or external type, not builtin)
			if BuiltinTypes[targetType] {
				v.addError(fieldPath, fmt.Sprintf("@relation target must be a content type, not builtin type %s", targetType))
				continue
			}
			// External types are valid relation targets (for templates with multiple schemas)
			if v.externalTypes[targetType] {
				// Skip further validation for external types - they're defined elsewhere
				continue
			}
			if !v.typeNames[targetType] {
				v.addError(fieldPath, fmt.Sprintf("@relation references unknown type: %s", targetType))
				continue
			}

			// Check for inverse field if specified
			if args, ok := field.Decorators[DecRelation].(map[string]any); ok {
				if inverse, ok := args["inverse"].(string); ok {
					// Find the inverse field in the target type
					found := false
					for _, targetTypeDef := range v.schema.Types {
						if targetTypeDef.Name != targetType {
							continue
						}
						for _, targetField := range targetTypeDef.Fields {
							if targetField.Name == inverse {
								found = true
								// Verify inverse field points back to this type
								if targetField.Type != typeDef.Name {
									v.addError(fieldPath, fmt.Sprintf("inverse field %s.%s does not reference type %s", targetType, inverse, typeDef.Name))
								}
								break
							}
						}
						break
					}
					if !found {
						v.addError(fieldPath, fmt.Sprintf("inverse field %s.%s does not exist", targetType, inverse))
					}
				}
			}
		}
	}
}

func (v *Validator) validateDefaultValue(fieldType string, value any) bool {
	switch fieldType {
	case TypeString, TypeText:
		_, ok := value.(string)
		return ok
	case TypeInt:
		_, ok := value.(int64)
		return ok
	case TypeFloat:
		switch value.(type) {
		case float64, int64:
			return true
		}
		return false
	case TypeBoolean:
		_, ok := value.(bool)
		return ok
	case TypeJSON:
		// JSON can be any type
		return true
	case TypeDateTime:
		// DateTime should be a string in ISO format
		_, ok := value.(string)
		return ok
	default:
		// Custom types - accept any value for now
		return true
	}
}

func (v *Validator) isPositiveInt(value any) bool {
	switch val := value.(type) {
	case int64:
		return val > 0
	case float64:
		return val > 0 && val == float64(int64(val))
	default:
		return false
	}
}

func (v *Validator) isNonNegativeInt(value any) bool {
	switch val := value.(type) {
	case int64:
		return val >= 0
	case float64:
		return val >= 0 && val == float64(int64(val))
	default:
		return false
	}
}

func (v *Validator) isNumber(value any) bool {
	switch value.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

func (v *Validator) addError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// ValidateSchema is a convenience function to validate a schema
func ValidateSchema(schema *Schema) error {
	validator := NewValidator(schema)
	errors := validator.Validate()

	if len(errors) > 0 {
		messages := make([]string, len(errors))
		for i, err := range errors {
			messages[i] = err.Error()
		}
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(messages, "\n  - "))
	}

	return nil
}

// ValidateSchemaWithExternalTypes validates a schema while treating specified external types as valid
// This is useful for templates where multiple schemas can reference each other
func ValidateSchemaWithExternalTypes(schema *Schema, externalTypes []string) error {
	validator := NewValidatorWithExternalTypes(schema, externalTypes)
	errors := validator.Validate()

	if len(errors) > 0 {
		messages := make([]string, len(errors))
		for i, err := range errors {
			messages[i] = err.Error()
		}
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(messages, "\n  - "))
	}

	return nil
}
