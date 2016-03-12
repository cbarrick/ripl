package lang

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Norm is the form to which unicode input is normalized.
const Norm = norm.NFD

// ASCIISymbols gives the valid ascii characters for a symbol functor.
const ASCIISymbols = "~`!@#$%^&*_-+=|\\:;\"<,>.?/"

// UnicodeSymbols gives the valid runes for a symbol functor.
var UnicodeSymbols = []*unicode.RangeTable{
	unicode.Symbol,
	unicode.Pc, // punctuation, connector
	unicode.Pd, // punctuation, dash
	unicode.Po, // punctuation, other (contains '!', and ',')
}

// ErrInvalidEnc is the error returned when the input cannot be lexed.
var ErrInvalidEnc = fmt.Errorf("invalid encoding")

// API
// --------------------------------------------------

// A Lexeme is a lexical item of Prolog.
type Lexeme struct {
	Typ  LexType
	Val  Value
	Tok  string
	Line int // zero-based line for the start of this token
	Col  int // zero-based column for the start of this token
}

// A LexType classifies types of lexeme.
type LexType int

// The types of lexeme.
const (
	LexErr LexType = iota // error

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

// Lex returns all of the tokens of the next clause.
func Lex(r io.Reader) <-chan Lexeme {
	ch := make(chan Lexeme, 4)
	go lex(r, ch)
	return ch
}

func (tok *Lexeme) String() string {
	return fmt.Sprintf("%q (%v)", tok.Tok, tok.Typ)
}

func (typ LexType) String() string {
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

// State Machine Infrastructure
// --------------------------------------------------

// A lexer contains the global state for the lexer state-machine.
type lexer struct {
	rd    *bufio.Reader
	ret   chan<- Lexeme // channel to return lexemes
	buf   *bytes.Buffer // buffer for current token
	cur   rune          // current rune, not yet in buf
	depth int           // number of unclosed parens etc.
	line  int           // zero-based line position of buf
	col   int           // zero-based column position of buf
	eof   bool
}

// lexStates encode the lexer state-machine. lexStates are functions that
// receive a pointer to the global state, modify that state, and return the next
// lexState of the machine.
//
// The machine halts when a lexState returns nil. Alternativly, if a lexState
// panics, a Lexeme of type LexErr is emitted, and the machine halts.
type lexState func(*lexer) lexState

// lex is the entry point of the lexing goroutine.
// It drives the state machine.
func lex(r io.Reader, ret chan<- Lexeme) {
	rd := bufio.NewReaderSize(Norm.Reader(r), 4)
	buf := new(bytes.Buffer)
	l := lexer{
		rd:  rd,
		ret: ret,
		buf: buf,
	}

	// At any point, the state machine may exit by panicing.
	// If so, a LexErr is emitted before closing.
	defer func() {
		err := recover()
		if err, ok := err.(error); ok {
			if err != nil {
				ret <- Lexeme{
					Typ:  LexErr,
					Tok:  err.Error(),
					Line: l.line,
					Col:  l.col,
				}
			}
		}
		close(ret)
	}()

	// prime the buffer
	l.read()

	// Under normal circumstances, the state-machine halts by returning nil.
	state := startState
	for state != nil {
		state = state(&l)
	}
}

// read consumes the next rune in the stream. The rune is added to the buffer.
func (l *lexer) read() rune {
	if l.cur != 0 {
		l.buf.WriteRune(l.cur)
	}

	if l.eof {
		l.cur = 0
		return 0
	}

	r, _, err := l.rd.ReadRune()
	if err == io.EOF {
		l.eof = true
	} else if err != nil {
		panic(err)
	}
	if r == '\uFFFD' {
		panic(ErrInvalidEnc)
	}
	l.cur = r
	return r
}

// readTo reads from the underlying reader up to the next occurence of delim.
func (l *lexer) readTo(delim rune) {
	var r rune
	for r != delim {
		r = l.read()
	}
}

// accept consumes the next rune if it is one of the given characters or belongs
// to one of the given unicode ranges. The return value is true if a character
// is consumed.
//
// It will never consume the '!' cut symbol or the ',' comma.
func (l *lexer) accept(chars string, ranges ...*unicode.RangeTable) (rune, bool) {
	var r = l.cur
	if r == '!' || r == ',' {
		return l.cur, false
	}
	if strings.ContainsRune(chars, r) || unicode.In(r, ranges...) {
		l.read()
		return l.cur, true
	}
	return l.cur, false
}

// acceptRun consumes as consecutive runes that are one of the given characters
// or belong to one of the given unicode ranges. The return value is true if any
// character is consumed.
//
// It will never consume the '!' cut symbol or the ',' comma.
func (l *lexer) acceptRun(chars string, ranges ...*unicode.RangeTable) (r rune, ok bool) {
	if r, ok = l.accept(chars, ranges...); !ok {
		return r, false
	}
	for ok {
		r, ok = l.accept(chars, ranges...)
	}
	return r, true
}

// Emit sends a lexeme to the user of the given type and value
// and flushes the buffer.
func (l *lexer) emit(typ LexType, val Value) {
	var dl, dc int // change in line/column over this token
	for r := range l.buf.String() {
		switch r {
		case '\r':
			dc = -l.col
		case '\n':
			dl++
			dc = -l.col
		default:
			dc++
		}
	}

	tok := l.buf.String()
	if val != nil {
		fmt.Fscan(l.buf, val)
	}
	l.ret <- Lexeme{typ, val, tok, l.line, l.col}

	l.line += dl
	l.col += dc
	l.buf.Reset()
}

// Prolog Lexer State Machine
// --------------------------------------------------
// This is the state machine to lex Prolog.

// the start state of the machine.
var startState lexState = lexAny

func lexAny(l *lexer) lexState {
	r := l.cur
	switch {

	// whitespace and comments
	case unicode.IsSpace(r):
		l.acceptRun(" \t\r\n", unicode.Space)
		l.emit(SpaceTok, nil)
		return lexAny
	case r == '%':
		l.readTo('\n')
		l.emit(CommentTok, nil)
		return lexAny

	// cuts, commas, and dots are special cases
	case r == '!':
		l.read()
		l.emit(FunctTok, new(Functor))
		return lexAny
	case r == ',':
		l.read()
		l.emit(FunctTok, new(Functor))
		return lexAny
	case r == '.':
		l.read()
		return emitDot

	// parens, brackets, and braces
	case strings.ContainsRune("([{}])", r):
		return lexParen

	// numbers may be preceeded by a negative
	case r == '-':
		l.read()
		r = l.cur
		if r < '0' || '9' < r {
			return lexSymbols
		}
		fallthrough
	case '0' < r && r < '9':
		return lexNumber

	// quoted tokens may contain escape characters
	case r == '"' || r == '\'':
		return lexQuote

	// variables always refer to the same value,
	// so we use the vars map to keep consistent identifiers
	case r == '_' || unicode.IsUpper(r):
		return lexVar

	// if it starts with a letter and is not a variable, it is a functor
	case unicode.IsLetter(r):
		return lexLetters

	// consecutive symbols are also functors
	case strings.ContainsRune(ASCIISymbols, r) || unicode.IsOneOf(UnicodeSymbols, r):
		return lexSymbols

	// auto-insert terminal at eof
	case r == 0:
		l.emit(TerminalTok, nil)
		return nil

	// all other runes are unacceptable
	default:
		panic(ErrInvalidEnc)
	}
}

func lexParen(l *lexer) lexState {
	r := l.cur
	l.read()
	switch r {
	case '(':
		l.depth++
		l.emit(ParenOpen, nil)
	case ')':
		l.depth--
		l.emit(ParenClose, nil)
	case '[':
		l.depth++
		l.emit(BracketOpen, nil)
	case ']':
		l.depth--
		l.emit(BracketClose, nil)
	case '{':
		l.depth++
		l.emit(BraceOpen, nil)
	case '}':
		l.depth--
		l.emit(BraceClose, nil)
	}
	return lexAny
}

func emitDot(l *lexer) lexState {
	if l.depth == 0 {
		r := l.cur
		if r == 0 || unicode.IsSpace(r) {
			l.emit(TerminalTok, nil)
			return nil
		}
	}
	return lexSymbols
}

func lexNumber(l *lexer) lexState {
	l.acceptRun("1234567890")
	_, a := l.accept(".")
	_, b := l.acceptRun("1234567890")
	if a && !b {
		l.emit(NumTok, new(Number))
		l.buf.WriteByte('.')
		return emitDot
	}
	l.accept("e")
	l.accept("+-")
	l.acceptRun("1234567890")
	l.emit(NumTok, new(Number))
	return lexAny
}

func lexVar(l *lexer) lexState {
	l.acceptRun("_", unicode.Letter)
	l.emit(VarTok, new(Variable))
	return lexAny
}

func lexLetters(l *lexer) lexState {
	l.acceptRun("_", unicode.Letter)
	l.emit(FunctTok, new(Functor))
	return lexAny
}

func lexSymbols(l *lexer) lexState {
	l.acceptRun(ASCIISymbols, UnicodeSymbols...)
	l.emit(FunctTok, new(Functor))
	return lexAny
}

func lexQuote(l *lexer) lexState {
	var quote = l.cur
	var typ LexType
	switch quote {
	case '\'':
		typ = FunctTok
	case '"':
		typ = StringTok
	}
	var r = l.read()
	for r != quote {
		r = l.read()
		if r == '\\' {
			r = l.read()
		}
	}
	l.read()
	l.emit(typ, new(Functor))
	return lexAny
}
