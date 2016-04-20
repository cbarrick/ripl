package lang

import (
	"fmt"
	"io"

	"github.com/cbarrick/ripl/lang/lex"
	"github.com/cbarrick/ripl/lang/ops"
	"github.com/cbarrick/ripl/lang/scope"
)

// A Parser iterates over the clauses of a Prolog source. Parsing happens
// place in parrallel with the main thread. The parser pauses after yielding
// a directive (:-/1), allowing synchronized access to the operator table and
// namespace.
type Parser struct {
	OpTab ops.Table        // operators to parse
	Scope *scope.Namespace // the symbol table, parsing may add new symbols
	Errs  []error          // any errors encountered are reported here

	lexer <-chan lex.Lexeme // main input
	ret   chan *Clause      // main output
	sync  chan struct{}     // used to pause after reading directives
	heap  []Subterm         // the clause is built onto this slice
	buf   lex.Lexeme        // the most recently read token
	args  [16]Subterm       // scratch space for parsing argument lists
}

const (
	heapSize   = 1024 // initial capacity of parser's heap
	bufferSize = 4    // initial output buffer size
)

// Parse creates a Parser over r.
func Parse(r io.Reader, optab ops.Table, sc *scope.Namespace) Parser {
	p := Parser{
		lexer: lex.Lex(r),
		ret:   make(chan *Clause, bufferSize),
		sync:  make(chan struct{}),
		OpTab: optab,
		Scope: sc,
		heap:  make([]Subterm, heapSize),
	}
	go p.run()
	return p
}

// Next returns the next clause or nil when the parser finished.
func (p *Parser) Next() *Clause {
	for {
		select {
		case <-p.sync:
			continue
		case c := <-p.ret:
			return c
		}
	}
}

// run is the entry point for the parser goroutine.
func (p *Parser) run() {
	neck := p.Scope.Name(lex.NewFunctor(":-"))
	for p.buf = range p.lexer {
		p.heap = p.heap[:0]
		t, _ := p.read(1200)
		p.heap = append(p.heap, t)

		c := Clause{
			Scope: p.Scope,
			heap:  make([]Subterm, len(p.heap)),
		}
		copy(c.heap, p.heap)
		p.ret <- &c

		if p.buf.Type != lex.TerminalTok {
			p.reportf("operator priority clash")
		}

		// pause after directives
		// this allows the caller to update the operator table, scope, etc
		if t.Key == neck && t.Arity == 1 {
			p.sync <- struct{}{}
		}
	}
	close(p.ret)
	close(p.sync)
}

// next reads the next Lexeme into the buffer.
func (p *Parser) advance() (tok lex.Lexeme) {
	tok = <-p.lexer
	p.buf = tok
	return tok
}

// skipSpace advances until the next non-space, non-comment token.
func (p *Parser) skipSpace() (tok lex.Lexeme) {
	tok = p.buf
	for tok.Type == lex.SpaceTok || tok.Type == lex.CommentTok {
		tok = <-p.lexer
	}
	p.buf = tok
	return tok
}

// reportf reports an error message.
// The line and column of the current token are prepended to the message.
func (p *Parser) reportf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	err := fmt.Errorf("%d:%d: %s", p.buf.Line+1, p.buf.Col, msg)
	p.Errs = append(p.Errs, err)
}

// read returns the next term with precidence no more than maxprec. It is the
// entry point of the parser, and is mutually recursive with the other read*
// methods. It is named after the read/1 predicate.
func (p *Parser) read(maxprec uint) (t Subterm, ok bool) {
	p.skipSpace()
	tok := p.buf
	switch tok.Type {
	default:
		t.Key = p.Scope.Name(tok.Symbol)
		p.advance()
		return t, true

	case lex.TerminalTok:
		return t, false

	case lex.LexErr:
		p.reportf(tok.String())
		return t, false

	case lex.FunctTok:
		t = p.readFunctor()
		return p.readOp(t, 0, maxprec), true

	case lex.ParenOpen:
		t = p.readGroup()
		return p.readOp(t, 0, maxprec), true

	case lex.BracketOpen:
		t = p.readList()
		return p.readOp(t, 0, maxprec), true
	}
}

func (p *Parser) readOp(lhs Subterm, lhsprec uint, maxprec uint) Subterm {
	if lhs.Atom() {
		str := p.Scope.Value(lhs.Key).String()
		for op := range p.OpTab.Get(str) {
			if maxprec < op.Prec {
				continue
			}

			prec := op.Prec
			switch op.Type {
			case ops.FX:
				prec--
				fallthrough
			case ops.FY:
				if rhs, ok := p.read(prec); ok {
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
		for op := range p.OpTab.Get(f.Symbol.String()) {
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
				p.advance()
			}

			prec := op.Prec
			switch op.Type {
			case ops.XF, ops.YF:
				t.Key = p.Scope.Name(f.Symbol)
				t.Arity = 1
				t.off = len(p.heap)
				p.heap = append(p.heap, lhs)
				return p.readOp(t, op.Prec, maxprec)
			case ops.XFX, ops.YFX:
				prec--
				fallthrough
			case ops.XFY:
				if rhs, ok := p.read(prec); ok {
					t.Key = p.Scope.Name(f.Symbol)
					t.Arity = 2
					t.off = len(p.heap)
					p.heap = append(p.heap, lhs, rhs)
					return p.readOp(t, op.Prec, maxprec)
				}
			}
		}
	}

	return lhs
}

func (p *Parser) readFunctor() (t Subterm) {
	k := p.Scope.Name(p.buf.Symbol)
	tok := p.advance()
	if tok.Type == lex.ParenOpen {
		args := p.readArgs()
		t.Key = k
		t.Arity = len(args)
		t.off = len(p.heap)
		for _, arg := range args {
			p.heap = append(p.heap, arg)
		}
	} else {
		t.Key = k
	}
	return t
}

func (p *Parser) readArgs() (args []Subterm) {
	args = p.args[:0]
	for {
		p.advance()
		arg, ok := p.read(999)
		if ok {
			args = append(args, arg)
		}
		switch {
		case p.buf.Type == lex.FunctTok && p.buf.Symbol.String() == ",":
			continue
		case p.buf.Type == lex.ParenClose:
			p.advance()
			return args
		default:
			p.reportf("expected ',' or ')', found %v", p.buf)
		}
	}
}

func (p *Parser) readGroup() (t Subterm) {
	p.advance() // consume open paren
	t, _ = p.read(1200)
	if p.buf.Type != lex.ParenClose {
		p.reportf("expected ')', found %v", p.buf)
	} else {
		p.advance() // consume close paren
	}
	return t
}

func (p *Parser) readList() (t Subterm) {
	panic("lists not implemented")
}
