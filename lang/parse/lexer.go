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
)

// API
// --------------------------------------------------

// The Lexer tokenizes and categorizes tokens in a prolog source, using an
// operator table to identify possible operators. The lexer runs in a separate
// goroutine and may read ahead of the user. The lexer pauses after it reads the
// clause terminator and resumes on demand when the next token is requested. It
// is not safe to update the operator table while the lexer is running.
// TODO: It would be a good idea to put a mutex on the operator table.
type Lexer struct {
	toks chan Token    // reads successive tokens
	ctrl chan struct{} // used for flow control in the goroutine
}

// Lex extracts Prolog lexemes from an io.Reader.
// The name of the lexer is used for error reports and the OpTable describes the
// available operators. The lexer may read ahead of the most recently returned
// lexeme, but it will not read past the current clause.
func Lex(input io.Reader, ops OpTable) Lexer {
	toks := make(chan Token, 2)
	ctrl := make(chan struct{})
	go lex(input, ops, toks, ctrl)
	return Lexer{
		toks,
		ctrl,
	}
}

// Read returns the next lexeme in the input.
func (l Lexer) Read() (tok Token, err error) {
	select {
	case <-l.ctrl:
		return l.Read()
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
	case _, ok := <-l.toks:
		if ok {
			l.Close()
		}
	case <-l.ctrl:
		l.Close()
	case l.ctrl <- struct{}{}:
		return
	}
}

// A Token is a lexical item.
// The lexer categorizes the token and adds position information.
type Token struct {
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

// A TokType identifies the kind of token represented by a Token.
type TokType int

const (
	// Special lexeme types
	EOF   TokType = iota // EOF indicator
	ERROR                // used to pass errors

	// Normal lexeme types
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
// Lexing technique taken from Rob Pike's presentation:
// Pike, Rob. "Lexical Scanning in Go". GTUG Sydney. 2011.
// http://cuddle.googlecode.com/hg/talk/lex.html

type lexState struct {
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

var syntaxErr = errors.New("lexing error")

// lex is the entry point of the lexer's goroutine
func lex(input io.Reader, ops OpTable, toks chan Token, ctrl chan struct{}) {
	s := lexState{
		input: input,
		buf:   make([]byte, bufSize),
		stack: make([]int, 0, 3),
		ops:   ops,
		toks:  toks,
		ctrl:  ctrl,
	}

	defer func() {
		err := recover()
		if err != nil && err != io.EOF {
			s.toks <- Token{
				err.(error).Error(),
				ERROR,
				s.lineNo + 1,
				s.colNo,
			}
		}
		close(toks)
	}()

	s.wait()
	for state := lexAny; state != nil; {
		select {
		case <-s.ctrl:
			return
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
		s.report(err)
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

// ignore skips over the pending text.
func (s *lexState) ignore() {
	s.start = s.pos
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

// acceptUntil consumes runes up to and including the next occurence of delim.
func (s *lexState) acceptUntil(delim rune) {
	for r := s.read(); r != delim; {
		r = s.read()
	}
	return
}

// emit sends the pending text as a Token of the given type.
func (s *lexState) emit(t TokType) {
	str := string(s.buf[s.start:s.pos])
	s.toks <- Token{str, t, s.lineNo + 1, s.colNo}
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

// wait blocks until the next call to Lexer.Next.
// This prevents the lexer from reading ahead of the parser.
func (s *lexState) wait() {
	s.ctrl <- struct{}{}
}

// Prolog Lexer State Machine
// --------------------------------------------------

// A stateFn is a state of the lexer, consuming runes and emiting lexemes
// and returning the next state of the lexer or nil when done.
type stateFn func(*lexState) stateFn

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
		s.acceptUntil('\n')
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

	// never read ahead past a dot
	if r == '.' {
		s.push()
		s.read()
		r = s.peek()
		if r == '%' || unicode.IsSpace(r) || r == eof {
			s.emit(EOC)
			s.wait()
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
		s.acceptUntil('\'')
		s.emit(IDENT)
		return lexAny
	}

	// consecutive lower case identifier
	if unicode.IsLower(r) {
		s.acceptRun("_", unicode.Letter, unicode.Number)
		s.emit(IDENT)
		return lexAny
	}

	s.report(syntaxErr)
	return nil
}

func lexVariable(s *lexState) stateFn {
	r := s.peek()
	if unicode.IsUpper(r) || r == '_' {
		for unicode.IsLetter(r) || r == '_' {
			s.read()
			r = s.peek()
		}
		s.emit(VAR)
		return lexAny
	}
	s.report(syntaxErr)
	return nil
}

func lexNumber(s *lexState) stateFn {
	r := s.peek()
	if '0' <= r && r <= '9' {
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
	s.report(syntaxErr)
	return nil
}
