package lang

import (
	"github.com/cbarrick/ripl/lang/scope"
	"github.com/cbarrick/ripl/lang/types"
)

// A Clause is a top-level term in Prolog source code.
// It supports both top-down and bottom-up traversal of its subterms.
type Clause []Subterm

// A Subterm is a component of a Clause.
type Subterm struct {
	Indicator
	off int // position within clause of first argument
}

// An Indicator is a functor/arity pair.
type Indicator struct {
	scope.Name
	Arity int
}

// Root returns the Subterm representing the root of the Clause.
func (c Clause) Root() Subterm {
	return c[len(c)-1]
}

// BottomUp returns an iterator over the Subterms of c in bottom-up order.
func (c Clause) BottomUp() <-chan Subterm {
	ch := make(chan Subterm, 1)
	go func(ch chan<- Subterm) {
		for i := range c {
			ch <- c[i]
		}
	}(ch)
	return ch
}

// TopDown returns an iterator over the Subterms of c in top-down order.
func (c Clause) TopDown() <-chan Subterm {
	ch := make(chan Subterm, 1)
	go func(ch chan<- Subterm) {
		queue := make([]Subterm, 0, len(c))
		queue = append(queue, c.Root())
		for len(queue) > 0 {
			ch <- queue[0]
			queue = append(queue[1:], c.args(queue[0])...)
		}
	}(ch)
	return ch
}

// args returns the arguments of t.
func (c Clause) args(t Subterm) []Subterm {
	return c[t.off : t.off+t.Arity]
}

// Atomic returns true if the arity of t is 0.
func (t Indicator) Atomic() bool {
	return t.Arity == 0
}

// Atom returns true if t represents an atom.
func (t Indicator) Atom() bool {
	return t.Arity == 0 && t.Name.Type == types.Funct
}
