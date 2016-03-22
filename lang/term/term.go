package term

import (
	"bytes"

	"github.com/cbarrick/ripl/lang/types"
)

// A Clause is a single term stored contiguously in bottom-up order.
type Clause struct {
	root Term
	heap []Term
}

// A Term is a Prolog term. A term is a syntax tree of functors and arguments.
type Term struct {
	id   int
	name Name
	args []Term
}

// Root returns the root term of the clause.
func (c Clause) Root() Term {
	return c.root
}

// Atomic returns true if t is not a compound term.
func (t Term) Atomic() bool {
	return len(t.args) == 0
}

// Atom returns true if t is an atom.
func (t Term) Atom() bool {
	return t.name.Typ == types.FunctorTyp && t.Atomic()
}

// String returns the canonical form of t.
func (t Term) String() string {
	var buf = new(bytes.Buffer)
	buf.WriteString(t.name.String())
	if len(t.args) > 0 {
		var open bool
		for _, arg := range t.args {
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
