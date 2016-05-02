package symbol

import (
	"fmt"
	"math/rand"
)

// A Name is assigned to each symbol in a Namespace. Functors are assigned
// unique names, and all other symbols are assigned a name reflecting the
// Prolog type of the symbol. The ordering of names reflects the relative order
// of symbols in the same namespace.
type Name float64

// Cmp compares names. Names representing symbols of different types can always be compared.
func (k Name) Cmp(c Name) float64 {
	return float64(k - c)
}

// Type identifies the Prolog type of a Symbol.
type Type int

// Prolog types, in order.
const (
	Funct Type = -iota
	Int
	Float
	Var
)

// A Namespace assigns Names to Symbols. Symbols are the literal symbols
// encountered by the lexer. Names are handles for Symbols that contain the
// minimum information for sorting, hashing, and unification. Names from
// separate Namespaces cannot be compared.
//
// Numeric types are keyed implicitly, and their values can be accessed and
// mutated directly from the Name. In most cases, this prevents the need to
// retrieve Numbers from the namespace.
type Namespace struct {
	heap *treap
}

// Neck returns the Name for the neck ":-" functor.
func (ns *Namespace) Neck() Name {
	return ns.Name(Functor(":-"))
}

// Name ensures that the Symbol is in the namespace, and returns its Name.
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

// Value retrieves the named Symbol from the namespace.
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
			addr = t.addr/2 + t.lo/2
			left = &treap{
				Symbol:   val,
				addr:     addr,
				lo:       root.lo,
				hi:       root.addr,
				priority: rand.Int63(),
			}
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
			addr = t.addr/2 + t.hi/2
			right = &treap{
				Symbol:   val,
				addr:     addr,
				lo:       root.addr,
				hi:       root.hi,
				priority: rand.Int63(),
			}
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

func (t *treap) String() string {
	if t == nil {
		return "_"
	}
	return fmt.Sprintf("(%v.%v %v %v)", t.Symbol, t.priority, t.left, t.right)
}
