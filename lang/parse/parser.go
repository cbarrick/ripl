package parse

import (
	"fmt"
	"io"
	"strings"

	"github.com/cbarrick/ripl/lang/term"
)

const (
	bufferSize = 128 // size of the initial lexeme buffer
)

// API
// --------------------------------------------------

// The Parser reads clauses from a Prolog source, using an operator table to
// identify operators. The parser must be closed to avoid goroutine leeks.
type Parser struct {
	l       Lexer                    // provides the Tokens
	buf     []Token                  // buffer of tokens
	pos     int                      // current position in the buffer
	stack   []int                    // pos history
	ops     OpTable                  // operators
	vars    map[string]term.Variable // var names
	lastVar term.Variable            // generates vars
	eof     bool                     // true after reading the eof
	err     *SyntaxError             // reported error(s)
}

// File creates a parser for the given reader.
func File(name string, input io.Reader, ops OpTable) Parser {
	return Parser{
		l:    Lex(name, input, ops),
		ops:  ops,
		vars: make(map[string]term.Variable),
	}
}

// String creates a parser for the given string.
func String(name string, input string, ops OpTable) Parser {
	return File(name, strings.NewReader(input), ops)
}

// Quick parses the string using the default operators and returns all clauses.
func Quick(str string) ([]term.Term, error) {
	return QuickOps(str, DefaultOps())
}

// QuickOps parses the string using the given operators and returns all clauses.
func QuickOps(str string, ops OpTable) (terms []term.Term, err error) {
	var t term.Term
	parser := String("string", str, ops)
	for t, err = parser.NextClause(); err == nil; {
		if t != nil {
			terms = append(terms, t)
		}
		t, err = parser.NextClause()
	}
	if err != io.EOF {
		return terms, nil
	}
	return terms, err
}

// NextClause returns the next clause in the input.
func (s *Parser) NextClause() (term.Term, error) {
	term, _ := s.readTerm(1200)
	tok := s.read()
	if tok.Typ == OP {
		return nil, priorityClash(tok)
	}
	if term == nil && s.err != nil {
		return nil, s.err
	}
	if term == nil && tok.Typ == EOF {
		return nil, io.EOF
	}
	if tok.Typ != EOC {
		return nil, unexpected(tok, EOC)
	}
	return term, nil
}

// Reset resets the parser to use the given input.
// It does not close the old underlying reader.
func (s *Parser) Reset(name string, input io.Reader, ops OpTable) {
	s.l.Close()
	s.l = Lex(name, input, ops)
	s.ops = ops
	s.buf = s.buf[:0]
	s.pos = 0
	s.stack = s.stack[:0]
	s.err = nil
	s.eof = false
}

// Close shuts down the Parser.
// It does not close the underlying reader.
func (s *Parser) Close() {
	s.l.Close()
}

// State Machine Infrastructure
// --------------------------------------------------
// The parser provides a helper interface for interacting with the underlying
// lexer. It provides a standard peek/read interface and a history stack for
// backtracking.

// report handles all errors.
func (s *Parser) report(err *SyntaxError) {
	err.Prev = s.err
	s.err = err
}

// peek returns the current token without advancing.
func (s *Parser) peek() Token {
	for len(s.buf) <= s.pos {
		tok, err := s.l.NextToken()
		if err != nil && err != io.EOF {
			s.report(wrapErr(tok, err))
		}
		s.buf = append(s.buf, tok)
	}
	return s.buf[s.pos]
}

// read returns the current token and advances the position.
func (s *Parser) read() Token {
	tok := s.peek()
	s.pos++
	return tok
}

// push saves the current position to the stack.
func (s *Parser) push() {
	s.stack = append(s.stack, s.pos)
}

// pop rewinds the current position from the stack.
func (s *Parser) pop() (pos int) {
	pos = s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return pos
}

// skipWhite advances past a run of whitespace and comments
func (s *Parser) skipWhite() (ok bool) {
	pos := s.pos
	for tok := s.peek(); tok.Typ == SPACE || tok.Typ == COMMENT; {
		s.pos++
		tok = s.peek()
	}
	return pos != s.pos
}

// Prolog Parser State Machine
// --------------------------------------------------
// Below is the parsing algorithm for Prolog. It is based on the operator
// precedence parser in the Edinburgh DEC-10 Prolog.
//
// The entry-point is readTerm(1200).
//
// http://www.j-paine.org/prolog/tools/files/contents.html

func unescape(ident string) string {
	// TODO
	return ident
}

// readTerm reads the longest term possible starting at the next token in the
// input such that the precedence of the term is no greater than maxprec.
func (s *Parser) readTerm(maxprec int) (t term.Term, prec int) {
	var lhs term.Term
	var lhsprec int

	s.skipWhite()
	tok := s.peek()
	switch tok.Typ {
	case OP:
		lhs, lhsprec = s.readOp(nil, 0, maxprec)
		if lhs == nil {
			lhs, lhsprec = s.readIdent()
		}
	case IDENT:
		lhs, lhsprec = s.readIdent()
	case VAR:
		lhs, lhsprec = s.readVar()
	case NUM:
		lhs, lhsprec = s.readNum()
	case GROUP_OPEN:
		lhs, lhsprec = s.readGroup()
	case LIST_OPEN:
		lhs, lhsprec = s.readList()
	case GROUP_CLOSE, LIST_CLOSE, EOC, EOF:
		return nil, maxprec
	case ERROR:
		return nil, maxprec
	default:
		panic("unknown token type")
	}

	s.skipWhite()
	return s.readOp(lhs, lhsprec, maxprec)
}

func (s *Parser) readIdent() (t term.Term, prec int) {
	var funct = s.read().Val
	var args []term.Term
	var next = s.peek()
	if next.Typ == GROUP_OPEN {
		s.read()
		for next.Typ != GROUP_CLOSE {
			t, _ = s.readTerm(999)
			s.skipWhite()
			if t == nil {
				break
			}
			args = append(args, t)
			next = s.read()
			if next.Val != "," {
				break
			}
		}
		if next.Typ != GROUP_CLOSE {
			s.report(unexpected(next, GROUP_CLOSE))
		}
	}
	t = term.Compound{
		Funct: unescape(funct),
		Args:  args,
	}
	return t, 0
}

func (s *Parser) readVar() (t term.Term, prec int) {
	tok := s.read()
	v := s.vars[tok.Val]
	if v == 0 {
		s.lastVar++
		v = s.lastVar
		s.vars[tok.Val] = v
	}
	t = v
	return t, 0
}

func (s *Parser) readNum() (t term.Term, prec int) {
	tok := s.read()
	var n term.Num
	_, err := fmt.Sscan(tok.Val, &n)
	if err != nil {
		s.report(wrapErr(tok, err))
	}
	t = n
	return t, 0
}

func (s *Parser) readGroup() (t term.Term, prec int) {
	tok := s.read()
	t, _ = s.readTerm(1200)
	switch tok = s.read(); {
	case tok.Typ == OP:
		s.report(priorityClash(tok))
	case tok.Typ != GROUP_CLOSE:
		s.report(unexpected(tok, GROUP_CLOSE))
	}
	return t, 0
}

func (s *Parser) readList() (t term.Term, prec int) {
	s.read()
	var args []term.Term
	for {
		arg, _ := s.readTerm(999)
		if arg != nil {
			args = append(args, arg)
		}

		s.skipWhite()
		switch next := s.read(); {
		case next.Val == "]":
			return term.List{args, nil}, 0

		case next.Val == "|":
			tail, _ := s.readTerm(1200)
			s.skipWhite()
			next = s.peek()
			if next.Val == "]" {
				s.read()
			} else {
				s.report(unexpected(next, LIST_CLOSE))
			}
			return term.List{args, tail}, 0

		case next.Val == ",":
			continue

		default:
			s.report(unexpected(next, LIST_CLOSE))
		}
	}
}

func (s *Parser) readOp(lhs term.Term, lhsprec, maxprec int) (t term.Term, prec int) {
	s.push()
	tok := s.read()

	switch tok.Typ {
	case OP:
		break
	case GROUP_CLOSE, LIST_CLOSE, EOC, EOF:
		s.pos = s.pop()
		return lhs, 0
	default:
		s.report(unexpected(tok, OP))
	}

	// find all apllicable operators
	ops := make(OpTable, 0, 2) // the maximum number of operator choices is two
	for _, op := range s.ops.Get(tok.Val) {
		var ok = true
		ok = ok && ((lhs == nil) == (op.Typ == FX || op.Typ == FY))
		ok = ok && (op.Prec <= maxprec)
		ok = ok && !((op.Typ == YFX) && !(lhsprec <= op.Prec))
		ok = ok && !((op.Typ == XFY) && !(lhsprec < op.Prec))
		ok = ok && !((op.Typ == XFX) && !(lhsprec < op.Prec))
		ok = ok && !((op.Typ == YF) && !(lhsprec <= op.Prec))
		ok = ok && !((op.Typ == XF) && !(lhsprec < op.Prec))
		if ok {
			ops = append(ops, op)
		}
	}

	for _, op := range ops {
		var opterm term.Term
		rhsprec := op.Prec
		switch op.Typ {

		case FX:
			rhsprec--
			fallthrough
		case FY:
			s.skipWhite()
			next := s.peek()
			if next.Typ == OP {
				nextops := s.ops.Get(next.Val)
				for _, nextop := range nextops {
					if nextop.Prec > op.Prec &&
						nextop.Typ != FX &&
						nextop.Typ != FX {
						s.report(ambiguousOp(next))
						s.pos = s.pop()
						return s.readIdent()
					}
				}
			}
			s.push()
			rhs, _ := s.readTerm(rhsprec)
			if rhs == nil {
				s.pos = s.pop()
				continue
			}
			s.pop()
			opterm = term.Compound{
				op.Name,
				[]term.Term{rhs},
			}

		case YFX, XFX:
			rhsprec--
			fallthrough
		case XFY:
			s.push()
			rhs, _ := s.readTerm(rhsprec)
			if rhs == nil {
				s.pos = s.pop()
				continue
			}
			s.pop()
			opterm = term.Compound{
				op.Name,
				[]term.Term{lhs, rhs},
			}

		case YF, XF:
			opterm = term.Compound{
				op.Name,
				[]term.Term{lhs},
			}

		default:
			panic("unknown operator type")
		}

		s.pop()
		s.skipWhite()
		return s.readOp(opterm, op.Prec, maxprec)
	}

	s.pos = s.pop()
	return lhs, 0
}
