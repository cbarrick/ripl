package lang

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// eof is used as a sentinal value for the state-machine.
const eof = '\x03'

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

// ErrInvalidEnc is the error returned when lexing invalid UTF-8.
var ErrInvalidEnc = fmt.Errorf("invalid encoding")

// API
// --------------------------------------------------

// A Lexeme is a lexical item of Prolog.
type Lexeme struct {
	Typ  LexType
	Val  interface{}
	Line uint // zero-based line for the start of this token
	Col  uint // zero-based column for the start of this token
}

// A LexType classifies types of lexeme.
type LexType int

// The types of lexeme.
// The Go type of a lexeme's value can be infered from its LexType.
const (
	LexErr LexType = iota // error

	SpaceTok    // string
	CommentTok  // string
	FunctTok    // string
	StringTok   // string
	NumTok      // math/big.Rat
	VarTok      // uint, 0 indicates the "_" placeholder
	ParenTok    // rune, includes parens, brackets, and braces
	TerminalTok // rune
)

// Lex returns all of the tokens of the next clause.
func Lex(r io.Reader) <-chan Lexeme {
	ch := make(chan Lexeme, 4)
	go lex(r, ch)
	return ch
}

func (tok *Lexeme) String() string {
	var format string
	switch tok.Typ {
	case LexErr:
		if tok.Val == nil {
			return "no more tokens"
		}
		return tok.Val.(error).Error()
	case SpaceTok, CommentTok, FunctTok, StringTok:
		format = "%q (%v)"
	case NumTok:
		format = "%v (%v)"
	case VarTok:
		format = "_%d (%v)"
	case ParenTok, TerminalTok:
		format = "%q (%v)"
	default:
		panic("unknown lexeme type")
	}
	return fmt.Sprintf(format, tok.Val, tok.Typ)
}

func (typ LexType) String() string {
	switch typ {
	case LexErr:
		return "LexErr"
	case SpaceTok:
		return "Space"
	case CommentTok:
		return "Comment"
	case FunctTok:
		return "Funct"
	case StringTok:
		return "String"
	case NumTok:
		return "Num"
	case VarTok:
		return "Var"
	case ParenTok:
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
	r       io.Reader
	ret     chan<- Lexeme     // channel to return lexemes
	vars    map[string]uint   // maintains ids for variables
	buf     bytes.Buffer      // buffer for current token
	runeBuf [utf8.UTFMax]byte // scratch space for decoding runes
	cur     rune              // current rune, not yet in buf
	depth   uint              // number of unclosed parens etc.
	line    uint              // zero-based line position of buf
	col     uint              // zero-based column position of buf
	eof     bool
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
	l := lexer{
		r:    Norm.Reader(r),
		ret:  ret,
		vars: make(map[string]uint),
	}

	// At any point, the state machine may exit by panicing.
	// If so, a LexErr is emitted before closing.
	defer func() {
		err := recover()
		if err == io.EOF {
			l.emit(TerminalTok, '.')
		} else if err != nil {
			l.emit(LexErr, err.(error))
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
func (l *lexer) read() (r rune) {
	if l.cur != 0 && l.cur != eof {
		l.buf.WriteRune(l.cur)
	}

	if l.eof {
		l.cur = eof
		return
	}

	var i int
	for !utf8.FullRune(l.runeBuf[:i]) {
		n, err := l.r.Read(l.runeBuf[i : i+1])
		i += n
		if err == io.EOF {
			l.eof = true
		} else if err != nil {
			panic(err)
		}
		if n == 0 {
			break
		}
	}
	if i > 0 {
		if utf8.Valid(l.runeBuf[:i]) {
			r, _ = utf8.DecodeRune(l.runeBuf[:i])
			l.cur = r
		} else {
			panic(ErrInvalidEnc)
		}
	} else {
		l.cur = eof
	}
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
func (l *lexer) accept(chars string, ranges ...*unicode.RangeTable) (ok bool) {
	var r = l.cur
	if r == '!' || r == ',' {
		return false
	}
	if strings.ContainsRune(chars, r) || unicode.In(r, ranges...) {
		l.read()
		return true
	}
	return false
}

// acceptRun consumes as consecutive runes that are one of the given characters
// or belong to one of the given unicode ranges. The return value is true if any
// character is consumed.
//
// It will never consume the '!' cut symbol or the ',' comma.
func (l *lexer) acceptRun(chars string, ranges ...*unicode.RangeTable) (ok bool) {
	for l.accept(chars, ranges...) {
		ok = true
	}
	return ok
}

// Emit sends a lexeme to the user of the given type and value
// and flushes the buffer.
func (l *lexer) emit(typ LexType, val interface{}) {
	l.ret <- Lexeme{typ, val, l.line, l.col}
	for i := l.buf.Len(); 0 < i; {
		r, size, err := l.buf.ReadRune()
		if err != nil {
			panic(err)
		}
		switch r {
		case '\r':
			l.col = 0
		case '\n':
			l.line++
			l.col = 0
		default:
			l.col++
		}
		i -= size
	}
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
		l.emit(SpaceTok, l.buf.String())
		return lexAny
	case r == '%':
		l.readTo('\n')
		l.emit(CommentTok, l.buf.String())
		return lexAny

	// cuts, commas, and dots are special cases
	case r == '!':
		l.read()
		l.emit(FunctTok, "!")
		return lexAny
	case r == ',':
		l.read()
		l.emit(FunctTok, ",")
		return lexAny
	case r == '.':
		return lexDot

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
	case r == eof:
		l.emit(TerminalTok, eof)
		return nil

	// all other runes are unacceptable
	default:
		panic(fmt.Errorf("unacceptable character %U", r))
	}
}

func lexParen(l *lexer) lexState {
	r := l.cur
	l.read()
	if strings.ContainsRune("([{", r) {
		l.depth++
	} else {
		l.depth--
	}
	l.emit(ParenTok, r)
	return lexAny
}

func lexDot(l *lexer) lexState {
	l.read()
	if l.depth == 0 {
		r := l.cur
		if r == eof || unicode.IsSpace(r) {
			l.emit(TerminalTok, '.')
			return nil
		}
	}
	return lexSymbols
}

func lexNumber(l *lexer) lexState {
	var num = new(big.Rat)
	l.acceptRun("1234567890")
	l.accept(".")
	l.acceptRun("1234567890")
	l.accept("e")
	l.accept("+-")
	l.acceptRun("1234567890")
	_, err := fmt.Sscan(l.buf.String(), num)
	if err != nil {
		panic(err)
	}
	l.emit(NumTok, num)
	return lexAny
}

func lexVar(l *lexer) lexState {
	l.acceptRun("_", unicode.Letter)
	name := l.buf.String()
	id := l.vars[name]
	if name != "_" && id == 0 {
		id = uint(len(l.vars) + 1)
		l.vars[name] = id
	}
	l.emit(VarTok, id)
	return lexAny
}

func lexLetters(l *lexer) lexState {
	l.acceptRun("_", unicode.Letter)
	l.emit(FunctTok, l.buf.String())
	return lexAny
}

func lexSymbols(l *lexer) lexState {
	l.acceptRun(ASCIISymbols, UnicodeSymbols...)
	l.emit(FunctTok, l.buf.String())
	return lexAny
}

func lexQuote(l *lexer) lexState {
	var quote = l.read()
	var typ LexType
	switch quote {
	case '\'':
		typ = FunctTok
	case '"':
		typ = StringTok
	}
	var r rune
	for r != quote {
		r = l.read()
		if r == '\\' {
			r = l.read()
		}
	}
	val, err := strconv.Unquote(l.buf.String())
	if err != nil {
		panic(err)
	}
	l.emit(typ, val)
	return lexAny
}
