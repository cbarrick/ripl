package term

import (
	"fmt"
	"io"

	"github.com/cbarrick/ripl/lang/oper"
)

// A parser contains the global state of the parsing algorithm.
type parser struct {
	lexer <-chan Lexeme // yields Lexemes to parse
	ops   oper.Table    // operators to parse
	ns    *Namespace    // the symbol table, parsing may add new symbols
	heap  []Term        // the clause is built onto this slice
	offs  map[int]int   // maps term id to offset of first argument
	buf   Lexeme        // the most recently read token
	args  [16]Term      // scratch space for parsing argument lists
	id    int           // generator for term ids
	errs  []error       // all errors encountered
}

// The default initial capacity of clause heaps
const defaultClauseSize = 32

// Parse reads a clause from r with respect to some operator table.
// Syntactically, a clause is a Prolog term followed by a period.
// The clause is built in bottom-up order.
func Parse(r io.Reader, ops oper.Table, ns *Namespace) (c Clause, errs []error) {
	// parse the term
	p := parser{
		lexer: Lex(r),
		ops:   ops,
		ns:    ns,
		heap:  make([]Term, 0, defaultClauseSize),
		offs:  make(map[int]int, defaultClauseSize),
	}
	p.next() // prime the buffer
	t, _ := p.readTerm(1200)
	c.heap = append(p.heap, t)

	// fix capacities and dangling references
	// all subterms must use the same heap and no argument slices may overlap
	for i, t := range c.heap {
		l := len(t.args)
		front := p.offs[t.id]
		back := p.offs[t.id] + l
		c.heap[i].args = c.heap[front:back:back]
	}
	c.root = c.heap[len(c.heap)-1]

	if p.buf.Typ != TerminalTok {
		p.reportf("operator priority clash")
	}
	return c, p.errs
}

// nextID generates a unique ID for every call.
func (p *parser) nextID() (id int) {
	id = p.id
	p.id++
	return id
}

// next reads the next Lexeme into the buffer.
func (p *parser) next() (tok Lexeme) {
	tok = <-p.lexer
	p.buf = tok
	return tok
}

// skip space advances until the next non-space token.
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

// Parser State Machine
// --------------------------------------------------

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
		t.id = p.nextID()
		t.name = p.ns.Name(tok.Val)
		p.next()
		return t, true

	case NumTok:
		t.id = p.nextID()
		t.name = p.ns.Name(tok.Val)
		p.next()
		return t, true

	case VarTok:
		t.id = p.nextID()
		t.name = p.ns.Name(tok.Val)
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
		for op := range p.ops.Get(lhs.name.String()) {
			if maxprec < op.Prec {
				continue
			}

			prec := op.Prec
			switch op.Typ {
			case oper.FX:
				prec--
				fallthrough
			case oper.FY:
				if rhs, ok := p.readTerm(prec); ok {
					off := len(p.heap)
					p.offs[lhs.id] = off
					p.heap = append(p.heap, rhs)
					lhs.args = p.heap[off:]
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
			} else if op.Typ == oper.XF || op.Typ == oper.XFX || op.Typ == oper.XFY {
				if op.Prec <= lhsprec {
					continue
				}
			} else if op.Typ == oper.YF || op.Typ == oper.YFX {
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
			case oper.XF, oper.YF:
				off := len(p.heap)
				p.heap = append(p.heap, lhs)
				t = Term{
					id:   p.nextID(),
					args: p.heap[off:],
					name: p.ns.Name(f.Val),
				}
				p.offs[t.id] = off
				return p.readOp(t, op.Prec, maxprec)
			case oper.XFX, oper.YFX:
				prec--
				fallthrough
			case oper.XFY:
				if rhs, ok := p.readTerm(prec); ok {
					off := len(p.heap)
					p.heap = append(p.heap, lhs, rhs)
					t = Term{
						id:   p.nextID(),
						args: p.heap[off:],
						name: p.ns.Name(f.Val),
					}
					p.offs[t.id] = off
					return p.readOp(t, op.Prec, maxprec)
				}
			}
		}
	}

	return lhs
}

func (p *parser) readFunctor() (t Term) {
	t.id = p.nextID()
	t.name = p.ns.Name(p.buf.Val)
	tok := p.next()
	if tok.Typ == ParenOpen {
		args := p.readArgs()
		off := len(p.heap)
		for _, arg := range args {
			p.heap = append(p.heap, arg)
		}
		p.offs[t.id] = off
		t.args = p.heap[off:]
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
