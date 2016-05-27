package symbol

import (
	"fmt"
	"math/rand"
)

// A Name is assigned to each symbol in a Namespace. Functors are assigned
// unique, positive names that reflect their relative ordering. All other
// symbols are assigned negative names equal to their Type.
type Name float64

// Cmp provides a partial ordering of Names. It returns a value less/greater
// than 0 if k is ordered before/after c. It returns 0 if k and c refer to the
// same symbol or if k and c cannot be compared.
//
// Names of symbols with different Types can always be compared, and Names of
// functors can be compared to any other Name. Comparing symbols of the same
// non-functor Type requires a lang.Indicator.
func (k Name) Cmp(c Name) float64 {
	return float64(k - c)
}

// Type identifies the Prolog type of a Symbol.
type Type int

// The types of Prolog symbols, in order.
const (
	Funct = -iota
	Int
	Float
	Var
)

// A Namespace stores a set of Symbols and assigns Names to them as they are
// inserted. The Names may then be used to retrieve the corresponding Symbol.
//
// Most important information about Symbols can be derived from the Name or
// higher level structures produced by the parser, and comparing Names is
// generally faster than comparing Symbols. However, Names from different
// namespaces cannot be compared.
type Namespace struct {
	heap *treap
}

// Neck is a convenience function to get the Name for the neck ":-" functor.
func (ns *Namespace) Neck() Name {
	return ns.Name(Functor(":-"))
}

// Name names a Symbol. If the Symbol has never been named, a new Name is
// generated and the Symbol is retained.
func (ns *Namespace) Name(val Symbol) Name {
	switch val := val.(type) {
	case Functor:
		var addr Name
		addr, ns.heap = ns.heap.address(val)
		return addr

	default:
		return Name(val.Type())
	}
}

// Value retrieves the named Symbol from the Namespace.
// If no such Symbol exists under that Name, it returns nil.
func (ns *Namespace) Value(k Name) Symbol {
	return ns.heap.get(k)
}

// A treap is a binary search tree using random priorities to maintain balance.
// See Wikipedia for a description: https://en.wikipedia.org/wiki/Treap
//
// The treap type is implemented as a persistent data-structures meaning a
// pointer to a treap will always represent the same tree. Thus operations that
// mutate the tree will return a pointer to a new root.
//
// This treap provides a positive float64 address key for each of their nodes.
type treap struct {
	Symbol
	addr        Name
	lo, hi      Name
	priority    int64
	left, right *treap
}

// The base controls how addresses are distributed. Addresses generated at the
// far right/left of the tree are set to their neighbor multiplied/divided by
// the base. Because of the way float64s are represented, a base of 2 results
// in an address space with very little entropy. A base of 31 appears to have
// better entropy to support hashing in the database.
const base = 31

// get retrieves a symbol from the treap, given its address.
func (t *treap) get(addr Name) Symbol {
	if t == nil {
		return nil
	}
	switch {
	case addr == t.addr:
		return t.Symbol
	case addr < t.addr:
		return t.left.get(addr)
	default:
		return t.right.get(addr)
	}
}

// address returns the address of a symbol. If the symbol does not yet have an
// address, it is retained and a suitable address is generated.
func (t *treap) address(val Symbol) (addr Name, root *treap) {
	if t == nil {
		return 1, &treap{
			Symbol:   val,
			addr:     1,
			lo:       0,
			hi:       2,
			priority: rand.Int63(),
		}
	}

	switch cmp := val.Cmp(t.Symbol); {
	case cmp == 0:
		return t.addr, t

	case cmp < 0:
		var left *treap
		if t.left == nil {
			root = new(treap)
			*root = *t
			left = newTreapLeft(t, val)
			addr = left.addr
			root.left = left
		} else {
			addr, left = t.left.address(val)
			if left == t.left {
				return addr, t
			}
			root = new(treap)
			*root = *t
			root.left = left
		}
		if left.priority > root.priority {
			left.right, root.left = root, left.right
			left.hi, root.lo = root.hi, left.addr
			root = left
		}
		return addr, root

	case cmp > 0:
		var right *treap
		if t.right == nil {
			root = new(treap)
			*root = *t
			right = newTreapRight(t, val)
			addr = right.addr
			root.right = right
		} else {
			addr, right = t.right.address(val)
			if right == t.right {
				return addr, t
			}
			root = new(treap)
			*root = *t
			root.right = right
		}
		if right.priority > root.priority {
			right.left, root.right = root, right.left
			right.lo, root.hi = root.lo, right.addr
			root = right
		}
		return addr, root
	}

	panic("unreachable")
}

func newTreapRight(t *treap, val Symbol) *treap {
	var addr Name
	if t.hi == 0 {
		addr = t.addr * base
	} else {
		addr = t.addr/2 + t.hi/2
	}
	return &treap{
		Symbol:   val,
		addr:     addr,
		lo:       t.addr,
		hi:       t.hi,
		priority: rand.Int63(),
	}
}

func newTreapLeft(t *treap, val Symbol) *treap {
	var addr Name
	if t.lo == 0 {
		addr = t.addr / base
	} else {
		addr = t.addr/2 + t.lo/2
	}
	return &treap{
		Symbol:   val,
		addr:     addr,
		lo:       t.lo,
		hi:       t.addr,
		priority: rand.Int63(),
	}
}

func (t *treap) String() string {
	if t == nil {
		return "_"
	}
	return fmt.Sprintf("(%v.%v %v %v)", t.Symbol, t.priority, t.left, t.right)
}
