package parse

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	. "github.com/cbarrick/ripl/lang"
)

const (
	eof       = rune(3) // EOF rune
	bufSize   = 64      // initial buffer size
	bufFactor = 8       // buffer grows by this much as needed, must be >= 4
	readAhead = 2       // max number of tokens to read ahead of client
)

// API
// --------------------------------------------------

// The Lexer tokenizes and categorizes tokens in a prolog source, using an
// operator table to identify possible operators. The lexer runs in a separate
// goroutine and may read ahead of the user. The lexer routine blocks after it
// reads the clause terminator and resumes on demand when the next token is
// requested. It is not safe to update the operator table while the lexer is
// running.
// TODO: It would be a good idea to put a mutex on the operator table.
type Lexer struct {
	toks chan Token    // reads successive tokens
	ctrl chan struct{} // used for flow control in the goroutine
}

// Lex constructs a Lexer for some source of Prolog lexemes.
// Only tokens in the operator table will be marked as operators.
// It is safe to modify the operator table until the first call to NextToken.
func Lex(name string, input io.Reader, ops OpTable) Lexer {
	toks := make(chan Token, readAhead)
	ctrl := make(chan struct{})
	go lex(name, input, ops, toks, ctrl)
	return Lexer{
		toks,
		ctrl,
	}
}

// NextToken returns the next lexeme in the input.
func (l Lexer) NextToken() (tok Token, err error) {
	select {
	case <-l.ctrl:
		return l.NextToken()
	case tok = <-l.toks:
		switch tok.Typ {
		case EOF:
			return tok, io.EOF
		case ERROR:
			return tok, errors.New(tok.Val)
		default:
			return tok, nil
		}
	}
}

// Close stops the lexer's goroutine.
func (l Lexer) Close() {
	select {
	case <-l.ctrl:
		l.Close()
	case _, ok := <-l.toks:
		if ok {
			l.Close()
		}
		return
	case l.ctrl <- struct{}{}:
		return
	}
}

// A Token is a lexical item.
// The lexer categorizes the token and adds position information.
type Token struct {
	Name   string
	Val    string
	Typ    TokType
	LineNo int
	ColNo  int
}

func (tok Token) String() string {
	switch tok.Typ {
	case EOF:
		return "EOF"
	case ERROR:
		return tok.Val
	default:
		return fmt.Sprintf("%q (%v)", tok.Val, tok.Typ)
	}
}

// A TokType classifies kinds of token.
type TokType int

const (
	// Special token types
	EOF   TokType = iota // EOF indicator
	ERROR                // used to pass errors

	// Normal token types
	OP          // operators
	IDENT       // atoms and functors
	VAR         // variable
	NUM         // number
	GROUP_OPEN  // open paren for a group
	GROUP_CLOSE // close paren for a group
	LIST_OPEN   // open brace for a list
	LIST_CLOSE  // close brace for a list
	SPACE       // whitespace
	COMMENT     // comment
	EOC         // end of clause
)

func (typ TokType) String() string {
	switch typ {
	case EOF:
		return "EOF"
	case ERROR:
		return "ERROR"
	case OP:
		return "operator"
	case IDENT:
		return "identifier"
	case VAR:
		return "variable"
	case NUM:
		return "number"
	case GROUP_OPEN:
		return "group"
	case GROUP_CLOSE:
		return "end of group"
	case LIST_OPEN:
		return "list"
	case LIST_CLOSE:
		return "end of list"
	case SPACE:
		return "whitespace"
	case COMMENT:
		return "comment"
	case EOC:
		return "end of clause"
	default:
		return "unknown"
	}
}

// State Machine Infrastructure
// --------------------------------------------------
// The lexer goroutine provides a state-machine interface for interfacing with
// the Reader. It reads bytes into a buffer and allows the transition functions
// to process to handle them one by one. When a token is emitted, the text
// consists of all the bytes that have been read since the previous emitted
// token.
//
// The states and transitions are programmed through state functions (type
// stateFn). A state functions is simply a function that process some bytes and
// returns the next stateFn.
//
// The machine interface also provides a history stack, allowing for
// backtracking. The history stack is discarded when a token is emitted.
//
// Lexing technique taken from Rob Pike's presentation:
// Pike, Rob. "Lexical Scanning in Go". GTUG Sydney. 2011.
// http://cuddle.googlecode.com/hg/talk/lex.html

// The lexState provides the machine interface.
//
// The control channel needs some explaining: When the goroutine must pause, it
// blocks by sending on the control channel. The next call to (Lexer).Read will
// read from the control channel to triger the resume. In the opposite direction
// (Lexer).Close writes to the control channel. When the goroutine reads this,
// it shuts down.
type lexState struct {
	name   string        // used for error messages
	input  io.Reader     // input being lexed
	buf    []byte        // the buffer
	size   int           // number of valid bytes in buffer
	start  int           // start position of this token
	pos    int           // current position in the buffer
	lineNo int           // number of the line being lexed (0 based)
	colNo  int           // offset within the current line
	stack  []int         // position history
	ops    OpTable       // set of operators to lex
	toks   chan Token    // channel of scanned tokens
	ctrl   chan struct{} // control channel
	eof    bool          // true after buffering the eof
}

// A stateFn is a state of the lexer, consuming runes and emiting lexemes
// and returning the next state of the lexer or nil when done.
type stateFn func(*lexState) stateFn

// lex is the entry point of the lexer's goroutine.
func lex(name string, input io.Reader, ops OpTable,
	toks chan Token, ctrl chan struct{}) {

	s := lexState{
		name:  name,
		input: input,
		buf:   make([]byte, bufSize),
		stack: make([]int, 0, 3),
		ops:   ops,
		toks:  toks,
		ctrl:  ctrl,
	}

	// Errors are handled by panicing the goroutine.
	// We emit a coresponding error token and continue.
	defer func() {
		err := recover()
		if err != nil && err != io.EOF {
			s.toks <- Token{
				s.name,
				err.(error).Error(),
				ERROR,
				s.lineNo + 1,
				s.colNo,
			}
		}
		lex(name, input, ops, toks, ctrl)
	}()

	// We wait until the first read, then start the main loop.
	s.wait()
	for state := startState; state != nil; {
		select {

		// cleanup
		case <-s.ctrl:
			close(toks)
			return

		// run the state machine
		default:
			state = state(&s)
		}
	}
}

// report handles all errors.
func (s *lexState) report(err error) {
	panic(err)
}

// buffer reads more text into the buffer.
func (s *lexState) buffer() bool {
	if s.eof {
		return false
	}

	// grow the buffer if needed
	if s.start == 0 {
		s.buf = append(s.buf, make([]byte, bufFactor)...)
	}

	// shift the current text to the start
	size := s.size - s.start
	copy(s.buf, s.buf[s.start:s.size])

	// keep the position stack consistent
	for i := range s.stack {
		s.stack[i] -= s.start
	}

	// read new input
	n, err := s.input.Read(s.buf[size:])
	s.size = size + n
	s.pos = s.pos - s.start
	s.start = 0
	if err == io.EOF {
		s.eof = true
		return n > 0
	}
	if err != nil {
		panic(err)
	}
	return true
}

// peek returns but does not consume the next rune in the input.
func (s *lexState) peek() rune {
	if s.pos >= s.size || !utf8.FullRune(s.buf[s.pos:]) {
		more := s.buffer()
		if !more {
			return eof
		}
	}
	r, _ := utf8.DecodeRune(s.buf[s.pos:])
	return r
}

// read returns and consumes the next rune in the input.
func (s *lexState) read() (r rune) {
	if s.pos >= s.size || !utf8.FullRune(s.buf[s.pos:]) {
		more := s.buffer()
		if !more {
			return eof
		}
	}
	r, width := utf8.DecodeRune(s.buf[s.pos:])
	s.pos += width
	return r
}

// pending returns the text of the pending token.
func (s *lexState) pending() string {
	return string(s.buf[s.start:s.pos])
}

// push saves the current position to the stack.
func (s *lexState) push() {
	s.stack = append(s.stack, s.pos)
}

// pop rewinds the current position from the stack.
func (s *lexState) pop() {
	pos := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	s.pos = pos
}

// accept consumes the next rune if it's from the valid set or unicode ranges.
func (s *lexState) accept(valid string, ranges ...*unicode.RangeTable) bool {
	r := s.peek()
	if strings.IndexRune(valid, r) >= 0 || unicode.In(r, ranges...) {
		s.read()
		return true
	}
	return false
}

// acceptRun consumes a run of runes from the valid set or unicode ranges.
func (s *lexState) acceptRun(valid string, ranges ...*unicode.RangeTable) bool {
	if s.accept(valid, ranges...) {
		r := s.peek()
		for strings.IndexRune(valid, r) >= 0 || unicode.In(r, ranges...) {
			s.read()
			r = s.peek()
		}
		return true
	}
	return false
}

// acceptUntil consumes runes runes from the ranges up-to but not including the
// next occurence of a rune in delim.
func (s *lexState) acceptUntil(delims string, ranges ...*unicode.RangeTable) bool {
	r := s.peek()
	if strings.IndexRune(delims, r) >= 0 || !unicode.In(r, ranges...) {
		return false
	}
	for strings.IndexRune(delims, r) < 0 && unicode.In(r, ranges...) {
		s.read()
		r = s.peek()
	}
	return true
}

// emit sends the pending text as a Token of the given type.
func (s *lexState) emit(t TokType) {
	str := s.pending()
	s.toks <- Token{s.name, str, t, s.lineNo + 1, s.colNo}
	s.start = s.pos
	s.stack = s.stack[:0]
	for _, r := range str {
		if r == '\n' {
			s.lineNo++
			s.colNo = 0
		} else {
			s.colNo++
		}
	}
}

// wait blocks until the next call to (Lexer).NextToken.
func (s *lexState) wait() {
	s.ctrl <- struct{}{}
}

// Prolog Lexer State Machine
// --------------------------------------------------
// This is the state machine to lex Prolog.

var startState = lexAny

func lexAny(s *lexState) stateFn {
	if s.acceptRun(" \n\t", unicode.White_Space) {
		s.emit(SPACE)
	}

	switch r := s.peek(); {
	case r == eof:
		s.emit(EOF)
		return nil

	case r == '(':
		s.read()
		s.emit(GROUP_OPEN)
		return lexAny
	case r == ')':
		s.read()
		s.emit(GROUP_CLOSE)
		return lexAny

	case r == '[':
		s.read()
		s.emit(LIST_OPEN)
		return lexAny
	case r == ']':
		s.read()
		s.emit(LIST_CLOSE)
		return lexAny

	case r == '%':
		s.acceptUntil("\n", unicode.PrintRanges...)
		s.emit(COMMENT)
		return lexAny

	case unicode.IsUpper(r) || r == '_':
		return lexVariable

	case '0' <= r && r <= '9':
		return lexNumber

	default:
		return lexText
	}
}

func lexText(s *lexState) stateFn {
	r := s.peek()

	// clause terminal
	if r == '.' {
		s.push()
		s.read()
		r = s.peek()
		if r == '%' || unicode.IsSpace(r) || r == eof {
			s.emit(EOC)
			s.wait() // must always wait after emitting EOC
			return lexAny
		}
		s.pop()
	}

	// operators
	// we search through the operators, prefering longer matches
	for _, op := range s.ops.ByLongest() {
		s.push()
		match := true
		i := 0
		for len(op.Name[i:]) > 0 {
			r, n := utf8.DecodeRuneInString(op.Name[i:])
			if r != s.read() {
				match = false
				break
			}
			i += n
		}
		if match {
			s.emit(OP)
			return lexAny
		}
		s.pop()
	}

	// single quoted identifier
	if r == '\'' {
		s.read()
		s.acceptUntil("'", unicode.PrintRanges...)
		r = s.read()
		if r != '\'' {
			s.report(fmt.Errorf("expected %q, found %q", '\'', r))
		}
		s.emit(IDENT)
		return lexAny
	}

	// consecutive letter identifier
	if unicode.IsLetter(r) {
		s.acceptRun("_", unicode.Letter, unicode.Number, unicode.Mark)
		s.emit(IDENT)
		return lexAny
	}

	// consecutive symbol identifier
	for {
		s.acceptUntil(".,", unicode.Punct, unicode.Symbol)
		r = s.peek()
		if r == '.' {
			s.push()
			s.read()
			r = s.peek()
			if r == '%' || unicode.IsSpace(r) || r == eof {
				s.pop()
				s.emit(IDENT)
				return lexText
			}
			continue
		}
		s.emit(IDENT)
		return lexAny
	}
}

func lexVariable(s *lexState) stateFn {
	s.acceptRun("_", unicode.Letter, unicode.Number, unicode.Mark)
	s.emit(VAR)
	return lexAny
}

func lexNumber(s *lexState) stateFn {
	s.acceptRun("0123456789")
	s.push()
	if s.accept(".") {
		if !s.acceptRun("0123456789") {
			s.pop()
		}
	}
	if s.accept("eE") {
		s.push()
		s.accept("+-")
		if !s.acceptRun("0123456789") {
			s.pop()
		}
	}
	s.emit(NUM)
	return lexAny
}
