package lang

import (
	"bytes"
	"fmt"
	"io"

	"github.com/cbarrick/ripl/lang/types"
)

// API
// --------------------------------------------------

// A Clause is a single term stored contiguously in bottom-up order.
type Clause []Term

// A Term is a Prolog term. A term is a syntax tree of functors and arguments.
type Term struct {
	Name Name
	Args []Term
}

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
	return t.Name.Typ == types.FunctorTyp && t.Atomic()
}

// String returns the canonical form of t.
func (t Term) String() string {
	var buf = new(bytes.Buffer)
	buf.WriteString(t.Name.String())
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

// Parse reads a clause from r with respect to some operator table.
// Syntactically, a clause is a Prolog term followed by a period.
// The clause is built in bottom-up order.
// The clause is returned along with the root term and any errors that may have occured.
func Parse(r io.Reader, ops OpTable, ns *Namespace) (Clause, Term, []error) {
	// parse the term
	p := parser{
		lexer: Lex(r),
		ops:   ops,
		ns:    ns,
	}
	p.next() // prime the buffer
	t, _ := p.readTerm(1200)
	c := Clause(append(p.heap, t))

	// fix argument capacities and ensure all subterms use the same storage
	// (the heap may have been reallocated during parsing)
	for i, t := range c {
		l := len(t.Args)
		c[i].Args = c[i-l : i:i]
	}
	t = c[len(c)-1]

	if p.buf.Typ != TerminalTok {
		p.reportf("operator priority clash")
	}
	return c, t, p.errs
}

// Parser
// --------------------------------------------------

type parser struct {
	lexer <-chan Lexeme
	ops   OpTable
	ns    *Namespace
	heap  []Term   // the clause is built onto this slice
	buf   Lexeme   // the most recently read token
	args  [16]Term // scratch space for parsing argument lists
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
		p.reportf(tok.Val.String())
		return t, false

	case FunctTok:
		return p.readFunctor(), true

	case StringTok:
		t.Name = p.ns.Name(tok.Val)
		p.next()
		return t, true

	case NumTok:
		t.Name = p.ns.Name(tok.Val)
		p.next()
		return t, true

	case VarTok:
		t.Name = p.ns.Name(tok.Val)
		p.next()
		return t, true

	case ParenOpen:
		return p.readGroup(), true
	case BracketOpen:
		return p.readList(), true

	case TerminalTok:
		return t, false

	default:
		p.reportf("cannont parse %v, not implemented", tok)
		return t, false
	}
}

func (p *parser) readOp(lhs Term, lhsprec uint, maxprec uint) Term {

	if lhs.Atom() {
		for op := range p.ops.Get(lhs.Name.String()) {
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
					return p.readOp(lhs, op.Prec, maxprec)
				}
			}
		}
	}

	var t Term
	f := p.skipSpace()
	var consumed bool
	if f.Typ == FunctTok {
		for op := range p.ops.Get(f.Val.String()) {
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
				t = Term{
					Args: p.heap[off:],
					Name: p.ns.Name(f.Val),
				}
				return p.readOp(t, op.Prec, maxprec)
			case XFX, YFX:
				prec--
				fallthrough
			case XFY:
				if rhs, ok := p.readTerm(prec); ok {
					off := len(p.heap)
					p.heap = append(p.heap, lhs, rhs)
					t = Term{
						Args: p.heap[off:],
						Name: p.ns.Name(f.Val),
					}
					return p.readOp(t, op.Prec, maxprec)
				}
			}
		}
	}

	return lhs
}

func (p *parser) readFunctor() (t Term) {
	t.Name = p.ns.Name(p.buf.Val)
	tok := p.next()
	if tok.Typ == ParenOpen {
		args := p.readArgs()
		off := len(p.heap)
		for _, arg := range args {
			p.heap = append(p.heap, arg)
		}
		t.Args = p.heap[off:]
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
		case p.buf.Typ == FunctTok && p.buf.Val.String() == ",":
			continue
		case p.buf.Typ == ParenClose:
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
	if p.buf.Typ != ParenClose {
		p.reportf("expected ')', found %v", p.buf)
	} else {
		p.next() // consume close paren
	}
	return t
}

func (p *parser) readList() (t Term) {
	panic("lists not implemented")
}
