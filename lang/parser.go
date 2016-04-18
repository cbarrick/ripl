package lang

import (
	"fmt"
	"io"

	"github.com/cbarrick/ripl/lang/lex"
	"github.com/cbarrick/ripl/lang/ops"
	"github.com/cbarrick/ripl/lang/scope"
)

// A parser contains the global state of the parsing algorithm.
type parser struct {
	lexer <-chan lex.Lexeme // yields Lexemes to parse
	optab ops.Table         // operators to parse
	ns    *scope.Namespace  // the symbol table, parsing may add new symbols
	heap  []Subterm         // the clause is built onto this slice
	buf   lex.Lexeme        // the most recently read token
	args  [16]Subterm       // scratch space for parsing argument lists
	id    int               // generator for term ids
	errs  []error           // all errors encountered
}

// The default initial capacity of clause heaps
const defaultClauseSize = 32

// Parse reads a clause from r with respect to some operator table.
// Syntactically, a clause is a Prolog term followed by a period.
// The clause is built in bottom-up order.
func Parse(r io.Reader, optab ops.Table, ns *scope.Namespace) (c Clause, errs []error) {
	p := parser{
		lexer: lex.Lex(r),
		optab: optab,
		ns:    ns,
		heap:  make([]Subterm, 0, defaultClauseSize),
	}
	p.next() // prime the buffer
	t, _ := p.readTerm(1200)

	c = Clause{
		Scope: p.ns,
		heap:  append(p.heap, t),
	}

	if p.buf.Type != lex.TerminalTok {
		p.reportf("operator priority clash")
	}
	return c, p.errs
}

// next reads the next Lexeme into the buffer.
func (p *parser) next() (tok lex.Lexeme) {
	tok = <-p.lexer
	p.buf = tok
	return tok
}

// skip space advances until the next non-space token.
func (p *parser) skipSpace() (tok lex.Lexeme) {
	tok = p.buf
	for tok.Type == lex.SpaceTok || tok.Type == lex.CommentTok {
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

func (p *parser) readTerm(maxprec uint) (t Subterm, ok bool) {
	if t, ok = p.read(); !ok {
		return t, false
	}
	t = p.readOp(t, 0, maxprec)
	return t, true
}

func (p *parser) read() (t Subterm, ok bool) {
	p.skipSpace()
	tok := p.buf

	switch tok.Type {
	case lex.LexErr:
		p.reportf(tok.String())
		return t, false

	case lex.FunctTok:
		return p.readFunctor(), true

	case lex.StringTok:
		t.Key = p.ns.Name(tok.Symbol)
		p.next()
		return t, true

	case lex.NumTok:
		t.Key = p.ns.Name(tok.Symbol)
		p.next()
		return t, true

	case lex.VarTok:
		t.Key = p.ns.Name(tok.Symbol)
		p.next()
		return t, true

	case lex.ParenOpen:
		return p.readGroup(), true
	case lex.BracketOpen:
		return p.readList(), true

	case lex.TerminalTok:
		return t, false

	default:
		p.reportf("cannont parse %v, not implemented", tok)
		return t, false
	}
}

func (p *parser) readOp(lhs Subterm, lhsprec uint, maxprec uint) Subterm {
	if lhs.Atom() {
		str := p.ns.Value(lhs.Key).String()
		for op := range p.optab.Get(str) {
			if maxprec < op.Prec {
				continue
			}

			prec := op.Prec
			switch op.Type {
			case ops.FX:
				prec--
				fallthrough
			case ops.FY:
				if rhs, ok := p.readTerm(prec); ok {
					lhs.off = len(p.heap)
					lhs.Arity = 1
					p.heap = append(p.heap, rhs)
					return p.readOp(lhs, op.Prec, maxprec)
				}
			}
		}
	}

	var t Subterm
	f := p.skipSpace()
	var consumed bool
	if f.Type == lex.FunctTok {
		for op := range p.optab.Get(f.Symbol.String()) {
			if maxprec < op.Prec {
				continue
			} else if op.Type == ops.XF || op.Type == ops.XFX || op.Type == ops.XFY {
				if op.Prec <= lhsprec {
					continue
				}
			} else if op.Type == ops.YF || op.Type == ops.YFX {
				if op.Prec < lhsprec {
					continue
				}
			} else {
				continue
			}

			if !op.Type.Prefix() && !consumed {
				p.next()
			}

			prec := op.Prec
			switch op.Type {
			case ops.XF, ops.YF:
				t = Subterm{
					Key:   p.ns.Name(f.Symbol),
					Arity: 1,
					off:   len(p.heap),
				}
				p.heap = append(p.heap, lhs)
				return p.readOp(t, op.Prec, maxprec)
			case ops.XFX, ops.YFX:
				prec--
				fallthrough
			case ops.XFY:
				if rhs, ok := p.readTerm(prec); ok {
					t = Subterm{
						Key:   p.ns.Name(f.Symbol),
						Arity: 2,
						off:   len(p.heap),
					}
					p.heap = append(p.heap, lhs, rhs)
					return p.readOp(t, op.Prec, maxprec)
				}
			}
		}
	}

	return lhs
}

func (p *parser) readFunctor() (t Subterm) {
	k := p.ns.Name(p.buf.Symbol)
	tok := p.next()
	if tok.Type == lex.ParenOpen {
		args := p.readArgs()
		t = Subterm{
			Key:   k,
			Arity: len(args),
			off:   len(p.heap),
		}
		for _, arg := range args {
			p.heap = append(p.heap, arg)
		}
	} else {
		t = Subterm{Key: k}
	}
	return t
}

func (p *parser) readArgs() (args []Subterm) {
	args = p.args[:0]
	for {
		p.next()
		arg, ok := p.readTerm(999)
		if ok {
			args = append(args, arg)
		}
		switch {
		case p.buf.Type == lex.FunctTok && p.buf.Symbol.String() == ",":
			continue
		case p.buf.Type == lex.ParenClose:
			p.next()
			return args
		default:
			p.reportf("expected ',' or ')', found %v", p.buf)
		}
	}
}

func (p *parser) readGroup() (t Subterm) {
	p.next() // consume open paren
	t, _ = p.readTerm(1200)
	if p.buf.Type != lex.ParenClose {
		p.reportf("expected ')', found %v", p.buf)
	} else {
		p.next() // consume close paren
	}
	return t
}

func (p *parser) readList() (t Subterm) {
	panic("lists not implemented")
}
