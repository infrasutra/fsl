package parser

import (
	"fmt"
	"unicode"
)

type Lexer struct {
	input        string
	position     int  // current position in input (current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char
	line         int
	column       int
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	tok := Token{
		Line:   l.line,
		Column: l.column,
	}

	switch l.ch {
	case 0:
		tok.Type = TokenEOF
		tok.Literal = ""
	case '{':
		tok.Type = TokenLBrace
		tok.Literal = string(l.ch)
	case '}':
		tok.Type = TokenRBrace
		tok.Literal = string(l.ch)
	case '[':
		tok.Type = TokenLBracket
		tok.Literal = string(l.ch)
	case ']':
		tok.Type = TokenRBracket
		tok.Literal = string(l.ch)
	case '(':
		tok.Type = TokenLParen
		tok.Literal = string(l.ch)
	case ')':
		tok.Type = TokenRParen
		tok.Literal = string(l.ch)
	case ':':
		tok.Type = TokenColon
		tok.Literal = string(l.ch)
	case ',':
		tok.Type = TokenComma
		tok.Literal = string(l.ch)
	case '!':
		tok.Type = TokenBang
		tok.Literal = string(l.ch)
	case '?':
		tok.Type = TokenQuestion
		tok.Literal = string(l.ch)
	case '@':
		tok.Type = TokenAt
		tok.Literal = string(l.ch)
	case '|':
		tok.Type = TokenPipe
		tok.Literal = string(l.ch)
	case '=':
		tok.Type = TokenEquals
		tok.Literal = string(l.ch)
	case '"':
		tok.Type = TokenString
		tok.Literal = l.readString()
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = l.lookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) || l.ch == '-' {
			tok.Literal = l.readNumber()
			tok.Type = TokenNumber
			return tok
		} else {
			tok.Type = TokenEOF
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		// Skip whitespace
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
			continue
		}

		// Skip line comments
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}

		// Skip block comments
		if l.ch == '/' && l.peekChar() == '*' {
			l.readChar() // consume /
			l.readChar() // consume *
			for {
				if l.ch == 0 {
					break
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar() // consume *
					l.readChar() // consume /
					break
				}
				l.readChar()
			}
			continue
		}

		break
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	if l.ch == '-' {
		l.readChar()
	}
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	l.readChar() // consume opening "
	position := l.position
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' && l.peekChar() == '"' {
			l.readChar() // consume \
		}
		l.readChar()
	}
	str := l.input[position:l.position]
	l.readChar() // consume closing "
	return str
}

func (l *Lexer) lookupIdent(ident string) TokenType {
	switch ident {
	case "type":
		return TokenTypeKeyword
	case "enum":
		return TokenEnumKeyword
	case "true", "false":
		return TokenBool
	default:
		return TokenIdent
	}
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}

// Error helpers
func (l *Lexer) Error(msg string) error {
	return fmt.Errorf("lexer error at line %d, column %d: %s", l.line, l.column, msg)
}
