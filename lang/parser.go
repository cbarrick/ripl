package lang

import (
	"bytes"
	"fmt"
	"io"
)

// API
// --------------------------------------------------

// A Clause is a single term stored contiguously in bottom-up order.
type Clause []Term

// A Term is a Prolog term. A term is a syntax tree of functors and arguments.
type Term struct {
	Typ  TermType
	Val  interface{}
	Args []Term
}

// TermType identifies a type of term (atom, number, etc).
type TermType int

// Types of term.
const (
	Structure TermType = iota
	Variable
	String
	Number
	List
)

// Root returns the root term of the clause.
func (c Clause) Root() Term {
	return c[len(c)-1]
}

// Atomic returns true if t is not a compound term.
func (t Term) Atomic() bool {
	return len(t.Args) == 0
}

// Atom returns true if t is an atom.
func (t Term) Atom() bool {
	return t.Typ == Structure && t.Atomic()
}

// String returns the canonical form of t.
func (t Term) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprint(t.Val))
	if len(t.Args) > 0 {
		var open bool
		for _, arg := range t.Args {
			if !open {
				buf.WriteRune('(')
				open = true
			} else {
				buf.WriteRune(',')
			}
			buf.WriteString(arg.String())
		}
		buf.WriteRune(')')
	}
	return buf.String()
}

// Parse reads a clause from r with respect to some operator table. Subterms are
// appended onto the heap in bottom-up level-order. The new heap slice is
// returned, and the backing array may be reallocated.
func Parse(r io.Reader, heap []Term, ops OpTable) (Clause, []Term, []error) {
	// parse the term
	var start = len(heap)
	p := parser{
		lexer: Lex(r),
		ops:   ops,
		heap:  heap,
		offs:  make(map[string]int, 16), // TODO: give this a default size
	}
	p.next() // prime the buffer
	t, _ := p.readTerm(1200)
	h := append(p.heap, t)
	c := Clause(h[start:])

	// ensure all subterms use the same storage
	// (the heap may have been reallocated during parsing)
	if len(heap) != 0 && &h[0] != &heap[0] {
		for i, sub := range c {
			off := p.offs[sub.String()]
			end := off + len(sub.Args)
			c[i].Args = h[off:end]
		}
	}

	if p.buf.Typ != TerminalTok {
		p.reportf("operator priority clash")
	}

	if len(p.errs) != 0 {
		return nil, nil, p.errs
	}

	return c, h, nil
}

// Parser
// --------------------------------------------------

type parser struct {
	lexer <-chan Lexeme
	ops   OpTable
	heap  []Term         // global storage for all term heaps
	offs  map[string]int // offsets of term heaps, keyed by canonical string
	buf   Lexeme         // the most recently read token
	args  [16]Term       // scratch space for parsing argument lists
	errs  []error
}

func (p *parser) next() (tok Lexeme) {
	tok = <-p.lexer
	p.buf = tok
	return tok
}

func (p *parser) skipSpace() (tok Lexeme) {
	tok = p.buf
	for tok.Typ == SpaceTok || tok.Typ == CommentTok {
		tok = p.next()
	}
	return tok
}

func (p *parser) reportf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	err := fmt.Errorf("%d:%d: %s", p.buf.Line+1, p.buf.Col, msg)
	p.errs = append(p.errs, err)
}

func (p *parser) readTerm(maxprec uint) (t Term, ok bool) {
	if t, ok = p.read(); !ok {
		return t, false
	}
	t = p.readOp(t, 0, maxprec)
	return t, true
}

func (p *parser) read() (t Term, ok bool) {
	p.skipSpace()
	tok := p.buf

	switch tok.Typ {
	case LexErr:
		p.reportf(tok.Val.(error).Error())
		return t, false

	case FunctTok:
		return p.readFunctor(), true

	case StringTok:
		t.Typ = String
		t.Val = tok.Val
		p.next()
		return t, true

	case NumTok:
		t.Typ = Number
		t.Val = tok.Val
		p.next()
		return t, true

	case VarTok:
		t.Typ = Variable
		t.Val = tok.Val
		p.next()
		return t, true

	case ParenTok:
		switch tok.Val.(rune) {
		case '(':
			return p.readGroup(), true
		case '[':
			return p.readList(), true
		default:
			return t, false
		}

	case TerminalTok:
		return t, false

	default:
		p.reportf("cannont parse %v, not implemented", tok)
		return t, false
	}
}

func (p *parser) readOp(lhs Term, lhsprec uint, maxprec uint) Term {

	if lhs.Atom() {
		for op := range p.ops.Get(lhs.Val.(string)) {
			if maxprec < op.Prec {
				continue
			}

			prec := op.Prec
			switch op.Typ {
			case FX:
				prec--
				fallthrough
			case FY:
				if rhs, ok := p.readTerm(prec); ok {
					off := len(p.heap)
					p.heap = append(p.heap, rhs)
					lhs.Args = p.heap[off:]
					p.offs[lhs.String()] = off
					return p.readOp(lhs, op.Prec, maxprec)
				}
			}
		}
	}

	var t Term
	f := p.skipSpace()
	var consumed bool
	if f.Typ == FunctTok {
		for op := range p.ops.Get(f.Val.(string)) {
			if maxprec < op.Prec {
				continue
			} else if op.Typ == XF || op.Typ == XFX || op.Typ == XFY {
				if op.Prec <= lhsprec {
					continue
				}
			} else if op.Typ == YF || op.Typ == YFX {
				if op.Prec < lhsprec {
					continue
				}
			} else {
				continue
			}

			if !op.Typ.Prefix() && !consumed {
				p.next()
			}

			prec := op.Prec
			switch op.Typ {
			case XF, YF:
				off := len(p.heap)
				p.heap = append(p.heap, lhs)
				t.Args = p.heap[off:]
				t.Val = f.Val.(string)
				p.offs[t.String()] = off
				return p.readOp(t, op.Prec, maxprec)
			case XFX, YFX:
				prec--
				fallthrough
			case XFY:
				if rhs, ok := p.readTerm(prec); ok {
					off := len(p.heap)
					p.heap = append(p.heap, lhs, rhs)
					t.Args = p.heap[off:]
					t.Val = f.Val.(string)
					p.offs[t.String()] = off
					return p.readOp(t, op.Prec, maxprec)
				}
			}
		}
	}

	return lhs
}

func (p *parser) readFunctor() (t Term) {
	t.Typ = Structure
	t.Val = p.buf.Val
	tok := p.next()
	switch tok.Typ {
	case ParenTok:
		if tok.Val.(rune) == '(' {
			args := p.readArgs()
			off := len(p.heap)
			for _, arg := range args {
				p.heap = append(p.heap, arg)
			}
			t.Args = p.heap[off:]
			p.offs[t.String()] = off
		}
	}
	return t
}

func (p *parser) readArgs() (args []Term) {
	args = p.args[:0]
	for {
		p.next()
		arg, ok := p.readTerm(999)
		if ok {
			args = append(args, arg)
		}
		switch {
		case p.buf.Typ == FunctTok && p.buf.Val.(string) == ",":
			continue
		case p.buf.Typ == ParenTok && p.buf.Val.(rune) == ')':
			p.next()
			return args
		default:
			p.reportf("expected ',' or ')', found %v", p.buf)
		}
	}
}

func (p *parser) readGroup() (t Term) {
	p.next() // consume open paren
	t, _ = p.readTerm(1200)
	if p.buf.Typ != ParenTok || p.buf.Val.(rune) != ')' {
		p.reportf("expected ')', found %v", p.buf)
	} else {
		p.next() // consume close paren
	}
	return t
}

func (p *parser) readList() (t Term) {
	panic("lists not implemented")
}
