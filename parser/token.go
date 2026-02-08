package parser

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString
	TokenNumber
	TokenBool
	TokenLBrace   // {
	TokenRBrace   // }
	TokenLBracket // [
	TokenRBracket // ]
	TokenLParen   // (
	TokenRParen   // )
	TokenColon    // :
	TokenComma    // ,
	TokenBang     // !
	TokenQuestion // ?
	TokenAt       // @
	TokenPipe     // | (for enum union types)
	TokenEquals   // = (for enum assignments)

	// Keywords
	TokenTypeKeyword // type keyword
	TokenEnumKeyword // enum keyword
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "IDENT"
	case TokenString:
		return "STRING"
	case TokenNumber:
		return "NUMBER"
	case TokenBool:
		return "BOOL"
	case TokenLBrace:
		return "{"
	case TokenRBrace:
		return "}"
	case TokenLBracket:
		return "["
	case TokenRBracket:
		return "]"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenColon:
		return ":"
	case TokenComma:
		return ","
	case TokenBang:
		return "!"
	case TokenQuestion:
		return "?"
	case TokenAt:
		return "@"
	case TokenPipe:
		return "|"
	case TokenEquals:
		return "="
	case TokenTypeKeyword:
		return "type"
	case TokenEnumKeyword:
		return "enum"
	default:
		return "UNKNOWN"
	}
}
