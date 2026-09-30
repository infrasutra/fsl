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
	enumNames     map[string]bool
	enumValues    map[string]map[string]bool
	fieldNames    map[string]map[string]bool
	relations     map[string][]string
	externalTypes map[string]bool
	libraryTypes  map[string]bool
	libraryEnums  map[string]bool
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
		libraryTypes:  make(map[string]bool),
		libraryEnums:  make(map[string]bool),
	}
}

// NewValidatorWithOptions creates a validator that knows about external types
// and the shared section library described by opts.
func NewValidatorWithOptions(schema *Schema, opts Options) *Validator {
	v := NewValidator(schema)
	for _, t := range opts.ExternalTypes {
		v.externalTypes[t] = true
	}
	if opts.Library != nil {
		for _, t := range opts.Library.Types {
			v.libraryTypes[t.Name] = true
		}
		for _, e := range opts.Library.Enums {
			v.libraryEnums[e.Name] = true
		}
	}
	return v
}

func (v *Validator) Validate() []ValidationError {

	for _, enumDef := range v.schema.Enums {
		if v.enumNames[enumDef.Name] {
			v.addError("", fmt.Sprintf("duplicate enum name: %s", enumDef.Name))
		}
		if v.libraryEnums[enumDef.Name] {
			v.addError("", fmt.Sprintf("enum '%s' is already defined in the section library", enumDef.Name))
		}
		v.enumNames[enumDef.Name] = true
		v.enumValues[enumDef.Name] = make(map[string]bool)
		for _, val := range enumDef.Values {
			v.enumValues[enumDef.Name][val] = true
		}
	}

	for _, typeDef := range v.schema.Types {
		if v.typeNames[typeDef.Name] {
			v.addError("", fmt.Sprintf("duplicate type name: %s", typeDef.Name))
		}
		if v.enumNames[typeDef.Name] {
			v.addError("", fmt.Sprintf("type name conflicts with enum name: %s", typeDef.Name))
		}
		if v.libraryTypes[typeDef.Name] {
			v.addError("", fmt.Sprintf("type '%s' is already defined in the section library", typeDef.Name))
		}
		v.typeNames[typeDef.Name] = true
		v.fieldNames[typeDef.Name] = make(map[string]bool)
	}

	for _, enumDef := range v.schema.Enums {
		v.validateEnumDef(&enumDef)
	}

	for i := range v.schema.Types {
		v.validateTypeDef(&v.schema.Types[i])
	}

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

	seen := make(map[string]bool)
	for _, val := range enumDef.Values {
		if seen[val] {
			v.addError(enumDef.Name, fmt.Sprintf("duplicate enum value: %s", val))
		}
		seen[val] = true
	}
}

func (v *Validator) validateTypeDef(typeDef *TypeDef) {

	if typeDef.Name == "" {
		v.addError("", "type name cannot be empty")
		return
	}

	if len(typeDef.Fields) == 0 {
		v.addError(typeDef.Name, "type must have at least one field")
	}

	for i := range typeDef.Fields {
		v.validateField(typeDef.Name, &typeDef.Fields[i])
	}
}

func (v *Validator) validateField(typeName string, field *FieldDef) {
	fieldPath := fmt.Sprintf("%s.%s", typeName, field.Name)

	if field.Name == "" {
		v.addError(fieldPath, "field name cannot be empty")
		return
	}

	if ReservedFieldNames[field.Name] {
		v.addError(fieldPath, fmt.Sprintf("field name '%s' is reserved", field.Name))
	}

	if v.fieldNames[typeName][field.Name] {
		v.addError(fieldPath, fmt.Sprintf("duplicate field name: %s", field.Name))
	}
	v.fieldNames[typeName][field.Name] = true

	if field.Type == "" {
		v.addError(fieldPath, "field type cannot be empty")
		return
	}

	isBuiltin := BuiltinTypes[field.Type]
	isEnum := v.enumNames[field.Type]
	isDefinedType := v.typeNames[field.Type]
	isExternalType := v.externalTypes[field.Type]
	isInlineEnum := field.Type == "Enum" && len(field.InlineEnum) > 0

	if !isBuiltin && !isEnum && !isDefinedType && !isExternalType && !isInlineEnum {
		v.addError(fieldPath, fmt.Sprintf("unknown type: %s", field.Type))
	}

	if (isDefinedType || isExternalType) && !isBuiltin && !isEnum && !isInlineEnum {
		field.IsRelation = true
	}

	if field.IsRelation {
		v.relations[typeName] = append(v.relations[typeName], field.Name)
	}

	if isInlineEnum {
		seen := make(map[string]bool)
		for _, val := range field.InlineEnum {
			if seen[val] {
				v.addError(fieldPath, fmt.Sprintf("duplicate inline enum value: %s", val))
			}
			seen[val] = true
		}
	}

	for decoratorName, decoratorValue := range field.Decorators {
		v.validateDecorator(fieldPath, field.Type, field, decoratorName, decoratorValue)
	}
}

func (v *Validator) validateDecorator(fieldPath, fieldType string, field *FieldDef, decoratorName string, decoratorValue any) {
	switch decoratorName {
	case DecMaxLength, DecMinLength:

		if fieldType != TypeString && fieldType != TypeText {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with String or Text types", decoratorName))
		}

		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, fmt.Sprintf("@%s value must be a positive integer", decoratorName))
		}

	case DecMin, DecMax:

		if fieldType != TypeInt && fieldType != TypeFloat {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with Int or Float types", decoratorName))
		}

		if !v.isNumber(decoratorValue) {
			v.addError(fieldPath, fmt.Sprintf("@%s value must be a number", decoratorName))
		}

	case DecPattern:

		if fieldType != TypeString {
			v.addError(fieldPath, "@pattern can only be used with String type")
		}

		if _, ok := decoratorValue.(string); !ok {
			v.addError(fieldPath, "@pattern value must be a string")
		}

	case DecDefault:

		if !v.validateDefaultValue(fieldType, decoratorValue) {
			v.addError(fieldPath, fmt.Sprintf("@default value type does not match field type %s", fieldType))
		}

	case DecUnique, DecIndex, DecSearchable, DecHidden:

	case DecRelation:

		v.validateRelationDecorator(fieldPath, field, decoratorValue)

	case DecSlices:

		if fieldType != TypeJSON {
			v.addError(fieldPath, "@slices can only be used with JSON type")
		}
		if field.Array {
			v.addError(fieldPath, "@slices cannot be used on array fields")
		}
		v.validateSlicesDecorator(fieldPath, decoratorValue)

	case DecMaxSize:

		if fieldType != TypeImage && fieldType != TypeFile {
			v.addError(fieldPath, "@maxSize can only be used with Image or File types")
		}
		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, "@maxSize value must be a positive integer (bytes)")
		}

	case DecFormats:

		if fieldType != TypeImage && fieldType != TypeFile {
			v.addError(fieldPath, "@formats can only be used with Image or File types")
		}
		v.validateFormatsDecorator(fieldPath, fieldType, decoratorValue)

	case DecPrecision:

		if fieldType != TypeFloat {
			v.addError(fieldPath, "@precision can only be used with Float type")
		}
		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, "@precision value must be a positive integer")
		}

	case DecMinItems:

		if !field.Array {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with array types", decoratorName))
		}
		if !v.isNonNegativeInt(decoratorValue) {
			v.addError(fieldPath, "@minItems value must be a non-negative integer")
		}

	case DecMaxItems:

		if !field.Array {
			v.addError(fieldPath, fmt.Sprintf("@%s can only be used with array types", decoratorName))
		}
		if !v.isPositiveInt(decoratorValue) {
			v.addError(fieldPath, "@maxItems value must be a positive integer")
		}

	case DecLabel, DecHelp, DecPlaceholder:
		if text, ok := decoratorValue.(string); !ok || strings.TrimSpace(text) == "" {
			v.addError(fieldPath, fmt.Sprintf("@%s needs a text value, e.g. @%s(\"...\")", decoratorName, decoratorName))
		}

	default:

		v.addError(fieldPath, fmt.Sprintf("unknown decorator: @%s", decoratorName))
	}
}

func (v *Validator) validateRelationDecorator(fieldPath string, field *FieldDef, decoratorValue any) {

	if decoratorValue == true {

		return
	}

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

		if !v.typeNames[targetType] && !v.libraryTypes[targetType] {
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

func (v *Validator) validateRelations() {
	for _, typeDef := range v.schema.Types {
		for _, field := range typeDef.Fields {
			if !field.IsRelation {
				continue
			}

			fieldPath := fmt.Sprintf("%s.%s", typeDef.Name, field.Name)
			targetType := field.Type

			if BuiltinTypes[targetType] {
				v.addError(fieldPath, fmt.Sprintf("@relation target must be a content type, not builtin type %s", targetType))
				continue
			}

			if v.externalTypes[targetType] {

				continue
			}
			if !v.typeNames[targetType] {
				v.addError(fieldPath, fmt.Sprintf("@relation references unknown type: %s", targetType))
				continue
			}

			if args, ok := field.Decorators[DecRelation].(map[string]any); ok {
				if inverse, ok := args["inverse"].(string); ok {

					found := false
					for _, targetTypeDef := range v.schema.Types {
						if targetTypeDef.Name != targetType {
							continue
						}
						for _, targetField := range targetTypeDef.Fields {
							if targetField.Name == inverse {
								found = true

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

		return true
	case TypeDateTime:

		_, ok := value.(string)
		return ok
	default:

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

// ValidateSchemaWithOptions validates a schema against the external types and
// section library described by opts.
func ValidateSchemaWithOptions(schema *Schema, opts Options) error {
	validator := NewValidatorWithOptions(schema, opts)
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
