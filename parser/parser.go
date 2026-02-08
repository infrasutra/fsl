package parser

import (
	"fmt"
	"strconv"
	"strings"
)

type Parser struct {
	lexer   *Lexer
	curTok  Token
	peekTok Token
	errors  []string
}

func NewParser(lexer *Lexer) *Parser {
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}
	// Read two tokens to initialize curTok and peekTok
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curTok = p.peekTok
	p.peekTok = p.lexer.NextToken()
}

func (p *Parser) ParseSchema() (*Schema, error) {
	schema := &Schema{
		Types: []TypeDef{},
		Enums: []EnumDef{},
	}

	for p.curTok.Type != TokenEOF {
		// Check for type-level decorators or keywords
		if p.curTok.Type == TokenAt || p.curTok.Type == TokenTypeKeyword {
			typeDef, err := p.parseTypeDef()
			if err != nil {
				return nil, err
			}
			schema.Types = append(schema.Types, *typeDef)
		} else if p.curTok.Type == TokenEnumKeyword {
			enumDef, err := p.parseEnumDef()
			if err != nil {
				return nil, err
			}
			schema.Enums = append(schema.Enums, *enumDef)
		} else {
			return nil, p.error(fmt.Sprintf("expected 'type', 'enum', or '@', got %s", p.curTok.Type))
		}
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors: %s", strings.Join(p.errors, "; "))
	}

	return schema, nil
}

// parseEnumDef parses a named enum definition
// enum Status { draft, published, archived }
func (p *Parser) parseEnumDef() (*EnumDef, error) {
	// Expect 'enum' keyword
	if p.curTok.Type != TokenEnumKeyword {
		return nil, p.error(fmt.Sprintf("expected 'enum' keyword, got %s", p.curTok.Type))
	}
	p.nextToken()

	// Expect enum name
	if p.curTok.Type != TokenIdent {
		return nil, p.error(fmt.Sprintf("expected enum name, got %s", p.curTok.Type))
	}
	enumDef := &EnumDef{
		Name:   p.curTok.Literal,
		Values: []string{},
	}
	p.nextToken()

	// Expect opening brace
	if p.curTok.Type != TokenLBrace {
		return nil, p.error(fmt.Sprintf("expected '{', got %s", p.curTok.Type))
	}
	p.nextToken()

	// Parse enum values
	for p.curTok.Type != TokenRBrace && p.curTok.Type != TokenEOF {
		if p.curTok.Type != TokenIdent {
			return nil, p.error(fmt.Sprintf("expected enum value, got %s", p.curTok.Type))
		}
		enumDef.Values = append(enumDef.Values, p.curTok.Literal)
		p.nextToken()

		// Optional comma separator
		if p.curTok.Type == TokenComma {
			p.nextToken()
		}
	}

	// Expect closing brace
	if p.curTok.Type != TokenRBrace {
		return nil, p.error(fmt.Sprintf("expected '}', got %s", p.curTok.Type))
	}
	p.nextToken()

	return enumDef, nil
}

func (p *Parser) parseTypeDef() (*TypeDef, error) {
	typeDef := &TypeDef{
		Decorators: []Decorator{},
		Fields:     []FieldDef{},
	}

	// Parse decorators before type keyword
	for p.curTok.Type == TokenAt {
		decorator, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}
		typeDef.Decorators = append(typeDef.Decorators, *decorator)
	}

	// Expect 'type' keyword
	if p.curTok.Type != TokenTypeKeyword {
		return nil, p.error(fmt.Sprintf("expected 'type' keyword, got %s", p.curTok.Type))
	}
	p.nextToken()

	// Expect type name
	if p.curTok.Type != TokenIdent {
		return nil, p.error(fmt.Sprintf("expected type name, got %s", p.curTok.Type))
	}
	typeDef.Name = p.curTok.Literal
	p.nextToken()

	// Expect opening brace
	if p.curTok.Type != TokenLBrace {
		return nil, p.error(fmt.Sprintf("expected '{', got %s", p.curTok.Type))
	}
	p.nextToken()

	// Parse fields
	for p.curTok.Type != TokenRBrace && p.curTok.Type != TokenEOF {
		field, err := p.parseFieldDef()
		if err != nil {
			return nil, err
		}
		typeDef.Fields = append(typeDef.Fields, *field)
	}

	// Expect closing brace
	if p.curTok.Type != TokenRBrace {
		return nil, p.error(fmt.Sprintf("expected '}', got %s", p.curTok.Type))
	}
	p.nextToken()

	return typeDef, nil
}

func (p *Parser) parseFieldDef() (*FieldDef, error) {
	field := &FieldDef{
		Decorators: make(map[string]any),
	}

	// Expect field name
	if p.curTok.Type != TokenIdent {
		return nil, p.error(fmt.Sprintf("expected field name, got %s", p.curTok.Type))
	}
	field.Name = p.curTok.Literal
	p.nextToken()

	// Expect colon
	if p.curTok.Type != TokenColon {
		return nil, p.error(fmt.Sprintf("expected ':', got %s", p.curTok.Type))
	}
	p.nextToken()

	// Parse field type (can be array, inline enum, or simple type)
	if p.curTok.Type == TokenLBracket {
		// Array type
		field.Array = true
		p.nextToken()

		// Parse element type (could be inline enum or regular type)
		if p.curTok.Type == TokenString {
			// Inline enum in array: [("draft" | "published")]
			inlineEnum, err := p.parseInlineEnum()
			if err != nil {
				return nil, err
			}
			field.InlineEnum = inlineEnum
			field.Type = "Enum"
		} else if p.curTok.Type != TokenIdent {
			return nil, p.error(fmt.Sprintf("expected type name, got %s", p.curTok.Type))
		} else {
			field.Type = p.curTok.Literal
			p.nextToken()
		}

		// Check if element is required
		if p.curTok.Type == TokenBang {
			field.Required = true
			p.nextToken()
		}

		// Expect closing bracket
		if p.curTok.Type != TokenRBracket {
			return nil, p.error(fmt.Sprintf("expected ']', got %s", p.curTok.Type))
		}
		p.nextToken()

		// Check if array itself is required
		if p.curTok.Type == TokenBang {
			field.ArrayReq = true
			p.nextToken()
		}
	} else if p.curTok.Type == TokenString {
		// Inline enum: "draft" | "published" | "archived"
		inlineEnum, err := p.parseInlineEnum()
		if err != nil {
			return nil, err
		}
		field.InlineEnum = inlineEnum
		field.Type = "Enum"

		// Check if required (must come after enum parsing)
		if p.curTok.Type == TokenBang {
			field.Required = true
			p.nextToken()
		}
	} else {
		// Simple type or named enum reference
		if p.curTok.Type != TokenIdent {
			return nil, p.error(fmt.Sprintf("expected type name, got %s", p.curTok.Type))
		}
		field.Type = p.curTok.Literal
		p.nextToken()

		// Check if required
		if p.curTok.Type == TokenBang {
			field.Required = true
			p.nextToken()
		}
	}

	// Parse decorators
	for p.curTok.Type == TokenAt {
		decorator, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}
		field.Decorators[decorator.Name] = p.decoratorArgsToValue(decorator.Args)

		// Mark as relation if @relation decorator is present
		if decorator.Name == DecRelation {
			field.IsRelation = true
		}
	}

	return field, nil
}

// parseInlineEnum parses inline enum values: "draft" | "published" | "archived"
func (p *Parser) parseInlineEnum() ([]string, error) {
	values := []string{}

	// First value must be a string
	if p.curTok.Type != TokenString {
		return nil, p.error(fmt.Sprintf("expected string for enum value, got %s", p.curTok.Type))
	}
	values = append(values, p.curTok.Literal)
	p.nextToken()

	// Parse additional values separated by |
	for p.curTok.Type == TokenPipe {
		p.nextToken() // consume |

		if p.curTok.Type != TokenString {
			return nil, p.error(fmt.Sprintf("expected string after '|', got %s", p.curTok.Type))
		}
		values = append(values, p.curTok.Literal)
		p.nextToken()
	}

	return values, nil
}

func (p *Parser) parseDecorator() (*Decorator, error) {
	// Expect @
	if p.curTok.Type != TokenAt {
		return nil, p.error(fmt.Sprintf("expected '@', got %s", p.curTok.Type))
	}
	p.nextToken()

	// Expect decorator name
	if p.curTok.Type != TokenIdent {
		return nil, p.error(fmt.Sprintf("expected decorator name, got %s", p.curTok.Type))
	}

	decorator := &Decorator{
		Name: p.curTok.Literal,
		Args: []any{},
	}
	p.nextToken()

	// Parse arguments if present
	if p.curTok.Type == TokenLParen {
		p.nextToken()

		for p.curTok.Type != TokenRParen && p.curTok.Type != TokenEOF {
			arg, err := p.parseDecoratorArg()
			if err != nil {
				return nil, err
			}
			decorator.Args = append(decorator.Args, arg)

			if p.curTok.Type == TokenComma {
				p.nextToken()
			}
		}

		if p.curTok.Type != TokenRParen {
			return nil, p.error(fmt.Sprintf("expected ')', got %s", p.curTok.Type))
		}
		p.nextToken()
	}

	return decorator, nil
}

func (p *Parser) parseDecoratorArg() (any, error) {
	switch p.curTok.Type {
	case TokenString:
		val := p.curTok.Literal
		p.nextToken()
		return val, nil
	case TokenNumber:
		val := p.curTok.Literal
		p.nextToken()
		// Try to parse as int first, then float
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			return intVal, nil
		}
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal, nil
		}
		return nil, p.error(fmt.Sprintf("invalid number: %s", val))
	case TokenBool:
		val := p.curTok.Literal == "true"
		p.nextToken()
		return val, nil
	case TokenIdent:
		// Named argument (key: value)
		key := p.curTok.Literal
		p.nextToken()
		if p.curTok.Type == TokenColon {
			p.nextToken()
			value, err := p.parseDecoratorArg()
			if err != nil {
				return nil, err
			}
			return map[string]any{key: value}, nil
		}
		// Just an identifier
		return key, nil
	default:
		return nil, p.error(fmt.Sprintf("unexpected token in decorator argument: %s", p.curTok.Type))
	}
}

func (p *Parser) decoratorArgsToValue(args []any) any {
	if len(args) == 0 {
		return true // Decorator with no args is like a flag
	}
	if len(args) == 1 {
		// Check if it's a map (named arg)
		if m, ok := args[0].(map[string]any); ok {
			// Flatten single-key map
			if len(m) == 1 {
				for _, v := range m {
					return v
				}
			}
		}
		return args[0]
	}
	// Multiple args - merge maps or return array
	merged := make(map[string]any)
	hasMap := false
	for _, arg := range args {
		if m, ok := arg.(map[string]any); ok {
			hasMap = true
			for k, v := range m {
				merged[k] = v
			}
		}
	}
	if hasMap {
		return merged
	}
	return args
}

func (p *Parser) error(msg string) error {
	errMsg := fmt.Sprintf("parser error at line %d, column %d: %s", p.curTok.Line, p.curTok.Column, msg)
	p.errors = append(p.errors, errMsg)
	return fmt.Errorf("%s", errMsg)
}
