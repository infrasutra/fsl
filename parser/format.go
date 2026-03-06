package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var decoratorOrder = []string{
	DecRequired,
	DecDefault,
	DecMinLength,
	DecMaxLength,
	DecMin,
	DecMax,
	DecPattern,
	DecUnique,
	DecIndex,
	DecSearchable,
	DecRelation,
	DecSlices,
	DecMinItems,
	DecMaxItems,
	DecFormats,
	DecMaxSize,
	DecPrecision,
	DecHidden,
	DecLabel,
	DecHelp,
	DecPlaceholder,
	DecCollection,
	DecSingleton,
	DecIcon,
	DecDescription,
}

var decoratorRank = makeDecoratorRank()

func makeDecoratorRank() map[string]int {
	rank := make(map[string]int, len(decoratorOrder))
	for i, name := range decoratorOrder {
		rank[name] = i
	}
	return rank
}

func Format(source string) (string, error) {
	lexer := NewLexer(source)
	p := NewParser(lexer)
	schema, err := p.ParseSchema()
	if err != nil {
		return "", err
	}
	return FormatSchema(schema), nil
}

func FormatSchema(schema *Schema) string {
	if schema == nil {
		return ""
	}

	sections := make([]string, 0, len(schema.Types)+len(schema.Enums))

	for _, typeDef := range schema.Types {
		sections = append(sections, formatTypeDef(typeDef))
	}

	for _, enumDef := range schema.Enums {
		sections = append(sections, formatEnumDef(enumDef))
	}

	out := strings.Join(sections, "\n\n")
	if out == "" {
		return ""
	}
	return out + "\n"
}

func formatTypeDef(typeDef TypeDef) string {
	var b strings.Builder

	decorators := sortDecorators(typeDef.Decorators)
	for _, decorator := range decorators {
		b.WriteString(formatDecorator(decorator.Name, decoratorArgsToCanonicalValue(decorator.Args)))
		b.WriteString("\n")
	}

	b.WriteString("type ")
	b.WriteString(typeDef.Name)
	b.WriteString(" {\n")

	for _, field := range typeDef.Fields {
		b.WriteString("  ")
		b.WriteString(formatField(field))
		b.WriteString("\n")
	}

	b.WriteString("}")
	return b.String()
}

func formatEnumDef(enumDef EnumDef) string {
	var b strings.Builder
	b.WriteString("enum ")
	b.WriteString(enumDef.Name)
	b.WriteString(" {\n")

	for i, value := range enumDef.Values {
		b.WriteString("  ")
		b.WriteString(value)
		if i < len(enumDef.Values)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	b.WriteString("}")
	return b.String()
}

func formatField(field FieldDef) string {
	var b strings.Builder
	b.WriteString(field.Name)
	b.WriteString(": ")
	b.WriteString(formatFieldType(field))

	for _, decorator := range sortedDecoratorEntries(field.Decorators) {
		b.WriteString(" ")
		b.WriteString(formatDecorator(decorator.Name, decorator.Value))
	}

	return b.String()
}

func formatFieldType(field FieldDef) string {
	if field.Array {
		elem := field.Type
		if len(field.InlineEnum) > 0 {
			elem = formatInlineEnum(field.InlineEnum)
		}

		if field.Required {
			elem += "!"
		}

		arrayType := "[" + elem + "]"
		if field.ArrayReq {
			arrayType += "!"
		}
		return arrayType
	}

	if len(field.InlineEnum) > 0 {
		enumType := formatInlineEnum(field.InlineEnum)
		if field.Required {
			enumType += "!"
		}
		return enumType
	}

	typ := field.Type
	if field.Required {
		typ += "!"
	}
	return typ
}

func formatInlineEnum(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Quote(value))
	}
	return strings.Join(parts, " | ")
}

func formatDecorator(name string, value any) string {
	if value == nil || value == true {
		return "@" + name
	}

	return fmt.Sprintf("@%s(%s)", name, formatDecoratorArg(value))
}

func formatDecoratorArg(value any) string {
	switch v := value.(type) {
	case string:
		return strconv.Quote(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, formatDecoratorArg(item))
		}
		return strings.Join(parts, ", ")
	case []string:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, strconv.Quote(item))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", key, formatDecoratorArg(v[key])))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func sortDecorators(decorators []Decorator) []Decorator {
	if len(decorators) == 0 {
		return nil
	}

	sorted := make([]Decorator, len(decorators))
	copy(sorted, decorators)

	sort.SliceStable(sorted, func(i, j int) bool {
		leftRank, leftKnown := decoratorRank[sorted[i].Name]
		rightRank, rightKnown := decoratorRank[sorted[j].Name]

		switch {
		case leftKnown && rightKnown:
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			return sorted[i].Name < sorted[j].Name
		case leftKnown:
			return true
		case rightKnown:
			return false
		default:
			return sorted[i].Name < sorted[j].Name
		}
	})

	return sorted
}

type decoratorEntry struct {
	Name  string
	Value any
}

func sortedDecoratorEntries(decorators map[string]any) []decoratorEntry {
	if len(decorators) == 0 {
		return nil
	}

	entries := make([]decoratorEntry, 0, len(decorators))
	for name, value := range decorators {
		entries = append(entries, decoratorEntry{Name: name, Value: value})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		leftRank, leftKnown := decoratorRank[entries[i].Name]
		rightRank, rightKnown := decoratorRank[entries[j].Name]

		switch {
		case leftKnown && rightKnown:
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			return entries[i].Name < entries[j].Name
		case leftKnown:
			return true
		case rightKnown:
			return false
		default:
			return entries[i].Name < entries[j].Name
		}
	})

	return entries
}

func decoratorArgsToCanonicalValue(args []any) any {
	if len(args) == 0 {
		return true
	}
	if len(args) == 1 {
		return args[0]
	}

	hasMap := false
	merged := make(map[string]any)
	for _, arg := range args {
		m, ok := arg.(map[string]any)
		if !ok {
			continue
		}
		hasMap = true
		for key, value := range m {
			merged[key] = value
		}
	}

	if hasMap {
		return merged
	}

	return args
}
