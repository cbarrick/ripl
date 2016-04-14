package term

import (
	"bytes"

	"github.com/cbarrick/ripl/lang/scope"
	"github.com/cbarrick/ripl/lang/types"
)

// A Clause is a top-level term in Prolog source code.
// It supports both top-down and bottom-up traversal of its subterms.
type Clause struct {
	Scope *scope.Namespace
	heap  []Subterm
}

// A Subterm is a component of a Clause. It contains a Key naming the functor of
// the term and an arity giving the number of arguments.
type Subterm struct {
	scope.Key
	Arity int
	off   int // position within clause of first argument
}

// Root returns the Subterm representing the root of the Clause.
func (c Clause) Root() Subterm {
	return c.heap[len(c.heap)-1]
}

// BottomUp returns an iterator over the Subterms of c in bottom-up order.
func (c Clause) BottomUp() <-chan Subterm {
	ch := make(chan Subterm, 1)
	go func(ch chan<- Subterm) {
		for i := range c.heap {
			ch <- c.heap[i]
		}
	}(ch)
	return ch
}

// TopDown returns an iterator over the Subterms of c in top-down order.
func (c Clause) TopDown() <-chan Subterm {
	ch := make(chan Subterm, 1)
	go func(ch chan<- Subterm) {
		queue := make([]Subterm, 0, len(c.heap))
		queue = append(queue, c.Root())
		for len(queue) > 0 {
			ch <- queue[0]
			queue = append(queue[1:], c.args(queue[0])...)
		}
	}(ch)
	return ch
}

// String returns the cannonical representation of the Clause.
//
// NOTE: This function may be removed in the future.
func (c Clause) String() string {
	// The only reason Clause embeds a pointer to its namespace is to support
	// this method. While this method is useful for testing, we may be able to
	// reduce GC overhead by removing pointers in our structs:
	// https://github.com/golang/go/wiki/CompilerOptimizations#non-scannable-objects
	buf := new(bytes.Buffer)
	var writeTerm func(Subterm)
	writeTerm = func(t Subterm) {
		buf.WriteString(c.Scope.Value(t.Key).String())
		if t.Arity == 0 {
			return
		}
		buf.WriteByte('(')
		for i, arg := range c.args(t) {
			writeTerm(arg)
			if i == t.Arity-1 {
				buf.WriteByte(')')
			} else {
				buf.WriteByte(',')
			}
		}
	}
	writeTerm(c.Root())
	return buf.String()
}

// args returns the arguments of t.
func (c Clause) args(t Subterm) []Subterm {
	return c.heap[t.off : t.off+t.Arity]
}

// Atomic returns true if the arity of t is 0.
func (t Subterm) Atomic() bool {
	return t.Arity == 0
}

// Atom returns true if t represents an atom.
func (t Subterm) Atom() bool {
	return t.Arity == 0 && t.Key.Type == types.Funct
}
