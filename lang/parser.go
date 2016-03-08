package lang

import (
	"bytes"
	"fmt"
	"io"
)

// API
// --------------------------------------------------

// A Term is a Prolog term. A term is a syntax tree of functors and arguments.
type Term struct {
	typ       TermType
	val       interface{}
	bootstrap [16]uint16 // initial storage for args
	args      []uint16   // indcies into heap
	heap      []Term     // storage for subterms
}

// TermType identifies a type of term (atom, number, etc).
type TermType uint

// Types of term.
const (
	Structure TermType = iota
	Variable
	String
	Number
	List
)

// Type returns the type of the term.
func (t *Term) Type() TermType {
	return t.typ
}

// Atomic returns true if t is not a compound term.
func (t *Term) Atomic() bool {
	return len(t.args) == 0
}

// Atom returns true if t is an atom.
func (t *Term) Atom() bool {
	return t.typ == Structure && t.Atomic()
}

// Size returns the number of structures in t.
func (t *Term) Size() int {
	return len(t.heap) + 1
}

// Parse reads a clause from r with respect to some operator table. Subterms are
// appended onto heap in bottom-up level-order, and a new slice is returned if
// the heap is reallocated.
func (t *Term) Parse(r io.Reader, ops OpTable, heap []Term) ([]Term, error) {
	var ok bool

	// parse the term
	p := parser{
		lexer: Lex(r),
		ops:   ops,
		heap:  t.heap[:0],
		offs:  make(map[string]int),
	}
	p.next() // prime the buffer
	*t, ok = p.readTerm(1200)

	// ensure all subterms use the same storage heap
	// (the heap may have been reallocated during parsing)
	t.heap = p.heap
	for i, sub := range p.heap {
		off := p.offs[sub.String()]
		end := off + len(sub.heap)
		p.heap[i].heap = p.heap[off:end]
	}

	if !ok {
		return heap, t.val.(error)
	}

	if p.buf.Typ != TerminalTok {
		return heap, fmt.Errorf("operator priority clash")
	}

	return p.heap, nil
}

// String returns the canonical string form of t.
func (t Term) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprint(t.val))
	if len(t.args) > 0 {
		var open bool
		for _, j := range t.args {
			if !open {
				buf.WriteRune('(')
				open = true
			} else {
				buf.WriteRune(',')
			}
			buf.WriteString(t.heap[j].String())
		}
		buf.WriteRune(')')
	}
	return buf.String()
}

// Parser Infrastructure
// --------------------------------------------------

type parser struct {
	lexer <-chan Lexeme
	ops   OpTable
	heap  []Term         // global storage for all term heaps
	offs  map[string]int // offsets of term heaps, keyed by canonical string
	buf   Lexeme         // the most recently read token
	args  [16]Term       // scratch space for parsing argument lists
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
	// TODO: better error handling
	panic(fmt.Errorf(format, args...))
}

// Prolog Parser
// --------------------------------------------------

func (p *parser) readTerm(maxprec uint) (t Term, ok bool) {
	if t, ok = p.read(); !ok {
		return t, false
	}
	t = p.readOp(t, 0, maxprec)
	return t, true
}

func (p *parser) readOp(lhs Term, lhsprec uint, maxprec uint) Term {
	var t Term
	f := p.skipSpace()

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

			prec := op.Prec
			switch op.Typ {
			case XFX, YFX:
				prec--
				fallthrough
			case XFY:
				p.next()
				off := p.offs[lhs.String()]
				if rhs, ok := p.readTerm(prec); ok {
					n := len(p.heap)
					t.args = t.bootstrap[:0]
					t.val = f.Val.(string)
					t.args = append(t.args, uint16(n-off), uint16(n+1-off))
					p.heap = append(p.heap, lhs, rhs)
					t.heap = p.heap[off:]
					p.offs[t.String()] = off
					return p.readOp(t, op.Prec, maxprec)
				}
				p.reportf("operator priority clash")
			}
		}
	}

	return lhs
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
		t.typ = String
		t.val = tok.Val
		p.next()
		return t, true

	case NumTok:
		t.typ = Number
		t.val = tok.Val
		p.next()
		return t, true

	case VarTok:
		t.typ = Variable
		t.val = tok.Val
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

func (p *parser) readFunctor() (t Term) {
	t.args = t.bootstrap[:0]
	t.typ = Structure
	t.val = p.buf.Val
	tok := p.next()
	switch tok.Typ {
	case ParenTok:
		if tok.Val.(rune) == '(' {
			off := len(p.heap)
			for _, arg := range p.readArgs() {
				t.args = append(t.args, uint16(len(p.heap)))
				p.heap = append(p.heap, arg)
			}
			t.heap = p.heap[off:]
			p.offs[t.String()] = off
		}
	}
	return t
}

func (p *parser) readArgs() (args []Term) {
	args = p.args[:0]
	for {
		p.next()
		arg, ok := p.readTerm(1000)
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
