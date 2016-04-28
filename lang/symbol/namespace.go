package symbol

import (
	"fmt"
	"math/rand"
)

// A Name names a symbol in a Namespace. Names contain the minimal information
// to compare named symbols without accessing the symbol itself.
//
// The address of a Name is used to retrieve the named symbol from the Namespace
// While the semantics of the address are well defined, its meaning is different
// for Symbols of different Prolog type:
//
// For functors, the address represents a lexicographic ordering of functors in
// the same namespace. For floats, the address is the value of the float. For
// integers, the address is always 0. And for variables, the address represents
// an arbitrary total ordering of variables.
//
// The hash of a name is equal to the hash of the named symbol. For integers,
// the hash is the value of the integer. For variables, the hash is always zero.
//
// The total ordering of names can be derived from the type, address, and hash.
type Name struct {
	Type Type
	Addr float64
	Hash int64
}

// Cmp compares named symbols. It is logically equivalent to calling Cmp on the
// symbols directly. Names from different namespaces cannot be compared.
func (k Name) Cmp(c Name) int {
	switch {
	// Prolog Types are enumerated in reverse sort order.
	case c.Type < k.Type:
		return -1
	case k.Type < c.Type:
		return +1

	// Ints are sorted by hash, everything else is sorted by address.
	case k.Type == Int:
		return int(k.Hash - c.Hash)
	case k.Addr < c.Addr:
		return -1
	case c.Addr < k.Addr:
		return +1
	default:
		return 0
	}
}

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
	neck *Name
}

// Neck returns the Name for the neck ":-" functor.
func (ns *Namespace) Neck() Name {
	if ns.neck == nil {
		k := ns.Name(Functor(":-"))
		ns.neck = &k
	}
	return *ns.neck
}

// Name ensures that the Symbol is in the namespace, and returns its Name.
func (ns *Namespace) Name(val Interface) Name {
	switch val := val.(type) {
	case *Number:
		if val.IsInt() {
			return Name{
				Type: Int,
				Addr: 0,
				Hash: val.Int64(),
			}
		}
		return Name{
			Type: Float,
			Addr: val.Float64(),
			Hash: val.Hash(),
		}

	case Functor:
		var addr float64
		addr, ns.heap = ns.heap.address(val)
		return Name{
			Type: Funct,
			Addr: addr,
			Hash: val.Hash(),
		}

	case Variable:
		var addr float64
		addr, ns.heap = ns.heap.address(val)
		return Name{
			Type: Var,
			Addr: addr,
			Hash: val.Hash(),
		}

	default:
		panic(fmt.Errorf("cannot name symbol: %v", val))
	}
}

// Value retrieves the named Symbol from the namespace.
func (ns *Namespace) Value(k Name) Interface {
	switch k.Type {
	case Int:
		val := new(Number)
		val.SetInt64(k.Hash)
		return val
	case Float:
		val := new(Number)
		val.SetFloat64(k.Addr)
		return val
	case Funct:
		return ns.heap.get(k.Addr)
	case Var:
		return ns.heap.get(k.Addr)
	default:
		panic(fmt.Errorf("unknown Prolog type for key: %v", k))
	}
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
	Interface
	addr        float64
	priority    int64
	lo, hi      float64
	left, right *treap
}

// get retrieves a symbol from the treap, given its address.
func (t *treap) get(addr float64) Interface {
	if t == nil {
		return nil
	}
	switch {
	case addr == t.addr:
		return t.Interface
	case addr < t.addr:
		return t.left.get(addr)
	default:
		return t.right.get(addr)
	}
}

// address returns the address of a symbol. If the symbol does not yet have an
// address, it is retained and a suitable address is generated.
func (t *treap) address(val Interface) (addr float64, root *treap) {
	if t == nil {
		return 1, &treap{
			Interface: val,
			addr:      1,
			priority:  rand.Int63(),
			lo:        0,
			hi:        2,
		}
	}

	switch val.Cmp(t.Interface) {
	case 0:
		return t.addr, t

	case -1:
		var left *treap
		if t.left == nil {
			root = new(treap)
			*root = *t
			addr = t.addr/2 + t.lo/2
			left = &treap{
				Interface: val,
				addr:      addr,
				lo:        root.lo,
				hi:        root.addr,
				priority:  rand.Int63(),
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

	case +1:
		var right *treap
		if t.right == nil {
			root = new(treap)
			*root = *t
			addr = t.addr/2 + t.hi/2
			right = &treap{
				Interface: val,
				addr:      addr,
				lo:        root.addr,
				hi:        root.hi,
				priority:  rand.Int63(),
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
	}

	return addr, root
}

func (t *treap) String() string {
	if t == nil {
		return "_"
	}
	return fmt.Sprintf("(%v.%v %v %v)", t.Interface, t.priority, t.left, t.right)
}
