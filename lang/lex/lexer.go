package lex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// API
// --------------------------------------------------

// A Lexeme is a lexical item of Prolog.
type Lexeme struct {
	Symbol
	Type
	Tok  string
	Line int
	Col  int
}

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

// Lex returns all of the tokens of the next clause.
func Lex(r io.Reader) <-chan Lexeme {
	ch := make(chan Lexeme, 4)
	go lex(r, ch)
	return ch
}

func (tok Lexeme) String() string {
	var typ string
	switch tok.Type {
	case LexErr:
		typ = "Lex Error"
	case SpaceTok:
		typ = "Whitespace"
	case CommentTok:
		typ = "Comment"
	case FunctTok:
		typ = "Functor"
	case StringTok:
		typ = "String"
	case NumTok:
		typ = "Number"
	case VarTok:
		typ = "Variable"
	case ParenOpen, ParenClose,
		BracketOpen, BracketClose,
		BraceOpen, BraceClose:
		typ = "Paren"
	case TerminalTok:
		typ = "Terminal"
	default:
		panic("unknown lexeme type")
	}
	return fmt.Sprintf("%q (%v)", tok.Tok, typ)
}

// State Machine Infrastructure
// --------------------------------------------------

// Norm is the form to which unicode input is normalized.
const Norm = norm.NFC

// ErrInvalidEnc is the error returned when the input cannot be lexed.
var ErrInvalidEnc = fmt.Errorf("input must be UTF-8")

// A lexer contains the global state for the lexer state-machine.
type lexer struct {
	rd    *bufio.Reader
	buf   *bytes.Buffer // buffer for current token
	ret   chan<- Lexeme // channel to return lexemes
	cur   rune          // current rune, not yet in buf
	depth int           // number of unclosed parens etc.
	line  int           // zero-based line position of buf
	col   int           // zero-based column position of buf
	eof   bool
}

// lexStates encode the lexer state-machine. lexStates are functions that
// receive a pointer to the global lexer state, modify that state, and return
// the next lexState of the machine.
//
// The machine halts when a lexState returns nil or panics.
type lexState func(*lexer) lexState

// lex is the entry point of the lexing goroutine.
// It drives the state machine.
func lex(r io.Reader, ret chan<- Lexeme) {
	l := lexer{
		rd:  bufio.NewReaderSize(Norm.Reader(r), 4),
		buf: new(bytes.Buffer),
		ret: ret,
	}

	// At any point, the state machine may exit by panicing.
	// If so, a LexErr is emitted before closing.
	defer func() {
		err := recover()
		if err, ok := err.(error); ok {
			if err != nil {
				ret <- Lexeme{
					Type: LexErr,
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
func (l *lexer) emit(typ Type, sym Symbol) {
	var tok = l.buf.String()
	var dl, dc int // change in line/column over this token
	for r := range tok {
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

	if sym != nil {
		_, err := fmt.Fscan(l.buf, sym)
		if err != nil {
			panic(err)
		}
	}
	l.ret <- Lexeme{sym, typ, tok, l.line + 1, l.col + 1}

	l.line += dl
	l.col += dc
	l.buf.Reset()
}

// Prolog Lexer State Machine
// --------------------------------------------------

// the start state of the machine.
var startState lexState = any

func any(l *lexer) lexState {
	r := l.cur
	switch {

	// whitespace and comments
	case unicode.IsSpace(r):
		l.acceptRun(" \t\r\n", unicode.Space)
		l.emit(SpaceTok, nil)
		return any
	case r == '%':
		l.readTo('\n')
		l.emit(CommentTok, nil)
		return any

	// cuts, commas, and dots are special cases
	case r == '!':
		l.read()
		l.emit(FunctTok, new(Functor))
		return any
	case r == ',':
		l.read()
		l.emit(FunctTok, new(Functor))
		return any
	case r == '.':
		l.read()
		return dot

	// parens, brackets, and braces
	case strings.ContainsRune("([{}])", r):
		return paren

	// numbers may be preceeded by a negative
	case r == '-':
		l.read()
		r = l.cur
		if r < '0' || '9' < r {
			return symbols
		}
		fallthrough
	case '0' < r && r < '9':
		return number

	// quoted tokens may contain escape characters
	case r == '\'':
		return quote

	// if it starts with an '_' underscore or an uppercase letter, it's a variable
	case r == '_' || unicode.IsUpper(r):
		return capital

	// if it starts with a letter and is not a variable, it's a functor
	case unicode.IsLetter(r):
		return lower

	// consecutive symbols are also functors
	case strings.ContainsRune(ASCIISymbols, r) || unicode.In(r, Symbols...):
		return symbols

	// auto-insert terminal at eof
	case l.eof:
		l.emit(TerminalTok, nil)
		return nil

	// all other runes are unacceptable
	default:
		panic(ErrInvalidEnc)
	}
}

func paren(l *lexer) lexState {
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
	return any
}

func dot(l *lexer) lexState {
	if l.depth == 0 {
		r := l.cur
		if r == 0 || unicode.IsSpace(r) {
			l.emit(TerminalTok, nil)
			return nil
		}
	}
	return symbols
}

func number(l *lexer) lexState {
	l.acceptRun("1234567890")
	_, a := l.accept(".")
	_, b := l.acceptRun("1234567890")
	if a && !b {
		l.emit(NumTok, new(Number))
		l.buf.WriteByte('.')
		return dot
	}
	l.accept("e")
	l.accept("+-")
	l.acceptRun("1234567890")
	l.emit(NumTok, new(Number))
	return any
}

func capital(l *lexer) lexState {
	l.acceptRun(ASCIILetters, Letters...)
	l.emit(VarTok, new(Variable))
	return any
}

func lower(l *lexer) lexState {
	l.acceptRun(ASCIILetters, Letters...)
	l.emit(FunctTok, new(Functor))
	return any
}

func symbols(l *lexer) lexState {
	l.acceptRun(ASCIISymbols, Symbols...)
	l.emit(FunctTok, new(Functor))
	return any
}

func quote(l *lexer) lexState {
	var r = l.read()
	for r != '\'' {
		r = l.read()
		if r == '\\' {
			r = l.read()
		}
	}
	l.read()
	l.emit(FunctTok, new(Functor))
	return any
}
