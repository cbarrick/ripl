package symbol

import (
	"fmt"
	"math"
	"math/rand"
)

// Type identifies the Prolog type of a Symbol.
type Type int

// The types of Prolog symbols, in standard sort order.
const (
	Var = iota
	Float
	Int
	Funct
)

// A Name is assigned to each symbol in a Namespace. The integer part of a Name
// gives the Type of the named Symbol and the fractional part gives the ordering
// of the named Symbol relative to other Symbols in the same Namespace. Names
// belonging to a Namespace are always positive.
//
// Names may also be generated at runtime, e.g. to name the result of an
// arithmetic evaluation or a generated variable. We call these dynamic Names.
// The Symbol refered to by a dynamic Name may not exist in any Namespace.
// The only requirement of dynamic Names is that the (positive) integer part of
// the name gives the Type. It is left to the user to interpret dynamic Names.
type Name float64

// NewName returns a new dynamic Name.
func NewName(t Type) Name {
	return Name(-t)
}

// Dynamic returns true when n is a dynamic Name, i.e. one generated at runtime.
func (n Name) Dynamic() bool {
	return n < 0
}

// Cmp provides a total ordering of Names. It returns a value less/greater than
// 0 if the Symbol named by n is ordered before/after the Symbol named by m. A
// value of 0 is returned if n and m refer to the same Symbol or if n and m are
// the same Type and either is dynamic.
//
// It is more efficient to compare Names directly when possible. I.e. prefer
// `n < m` over `n.Cmp(m)`. This is only possible when neither n nor m are
// dynamic.
func (n Name) Cmp(m Name) float64 {
	if n < 0 || m < 0 {
		if n.Type() == m.Type() {
			return 0
		}
		n = Name(math.Abs(float64(n)))
		m = Name(math.Abs(float64(m)))
	}
	return float64(n - m)
}

// Type returns the type of the named symbol.
func (n Name) Type() Type {
	return Type(math.Abs(float64(n)))
}

// A Namespace stores a set of Symbols and assigns Names to them as they are
// inserted. The Names may then be used to retrieve the corresponding Symbol.
//
// Most important information about Symbols can be derived from the Name or
// higher level structures produced by the parser, and comparing Names is
// generally faster than comparing Symbols. However, Names from different
// namespaces cannot be compared.
type Namespace struct {
	spaces [4]*treap // one address space for each Type
}

// Neck is a convenience function to get the Name for the neck ":-" functor.
func (ns *Namespace) Neck() Name {
	return ns.Name(Functor(":-"))
}

// Name names a Symbol. If the Symbol has never been named, a new Name is
// generated and the Symbol is retained.
func (ns *Namespace) Name(val Symbol) Name {
	t := val.Type()
	var addr float64
	addr, ns.spaces[t] = ns.spaces[t].address(val)
	return Name(addr + float64(t))
}

// Value retrieves the named Symbol from the Namespace.
// If no such Symbol exists under that Name, it returns nil.
func (ns *Namespace) Value(k Name) Symbol {
	t := k.Type()
	return ns.spaces[t].get(float64(k) - float64(t))
}

// A treap is a binary search tree using random priorities to maintain balance.
// See Wikipedia for a description: https://en.wikipedia.org/wiki/Treap.
//
// The treap type is implemented as a path-copying persistent tree. A pointer to
// a treap will always represent the same data. Operations that mutate the
// treap will return a pointer to a new root node.
//
// This treap provides a float64 address for each node. The address is between
// 0 and 1 and the relative order of addresses reflects the relative order of
// the symbols. Addresses are generated with entropy, and thus serve as a kind
// of hash code for Symbols.
//
// See https://gist.github.com/cbarrick/67adf9fdb4e884ae514de56c164294b2.
type treap struct {
	Symbol
	addr        float64
	lo, hi      float64
	priority    int64
	left, right *treap
}

// The weight controls how addresses are distributed. Because of the way
// float64s are represented, a weight of 1/2 results in am address space with
// little entropy. A weight of 1/3 performs better in that regard.
const weight = 1.0 / 3.0

// get retrieves a symbol from the treap, given its address.
func (t *treap) get(addr float64) Symbol {
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
func (t *treap) address(val Symbol) (addr float64, root *treap) {
	if t == nil {
		return 0.5, &treap{
			Symbol:   val,
			addr:     0.5,
			lo:       0,
			hi:       1,
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

	default:
		panic("unreachable")
	}
}

func newTreapRight(t *treap, val Symbol) *treap {
	return &treap{
		Symbol:   val,
		addr:     t.addr*weight + t.hi*(1-weight),
		lo:       t.addr,
		hi:       t.hi,
		priority: rand.Int63(),
	}
}

func newTreapLeft(t *treap, val Symbol) *treap {
	return &treap{
		Symbol:   val,
		addr:     t.addr*weight + t.lo*(1-weight),
		lo:       t.lo,
		hi:       t.addr,
		priority: rand.Int63(),
	}
}

func (t *treap) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("(%v@%v %v %v)", t.Symbol, t.priority, t.left, t.right)
}