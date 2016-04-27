package lexer

// A Type classifies types of lexeme.
type Type int

// The types of lexeme.
const (
	LexErr Type = iota
	SpaceTok
	CommentTok
	FunctTok
	StringTok
	NumTok
	VarTok
	ParenOpen
	ParenClose
	BracketOpen
	BracketClose
	BraceOpen
	BraceClose
	TerminalTok
)

func (typ Type) String() string {
	switch typ {
	case LexErr:
		return "Lex Error"
	case SpaceTok:
		return "Whitespace"
	case CommentTok:
		return "Comment"
	case FunctTok:
		return "Functor"
	case StringTok:
		return "String"
	case NumTok:
		return "Number"
	case VarTok:
		return "Variable"
	case ParenOpen, ParenClose,
		BracketOpen, BracketClose,
		BraceOpen, BraceClose:
		return "Paren"
	case TerminalTok:
		return "Terminal"
	default:
		panic("unknown lexeme type")
	}
}
