package scope

import (
	"fmt"
	"math/rand"

	"github.com/cbarrick/ripl/lang/lex"
	"github.com/cbarrick/ripl/lang/types"
)

// A Key names a symbol in a Namespace. Keys contain the minimal information
// to compare named symbols without accessing the symbol itself.
//
// The address of a Key is used to retrieve the named symbol from the Namespace
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
type Key struct {
	NSID int64
	Type types.PLType
	Addr float64
	Hash int64
}

// Cmp compares named symbols. It is logically equivalent to calling Cmp on the
// symbols directly. Keys from different namespaces cannot be compared.
func (k Key) Cmp(c Key) int {
	switch {
	case k.NSID != c.NSID:
		panic("cannot compare names from different namespaces (NSID mismatch)")

	// PLTypes are enumerated in reverse sort order.
	case c.Type < k.Type:
		return -1
	case k.Type < c.Type:
		return +1

	// Ints are sorted by hash, everything else is sorted by address.
	case k.Type == types.Int:
		return int(k.Hash - c.Hash)
	case k.Addr < c.Addr:
		return -1
	case c.Addr < k.Addr:
		return +1
	default:
		return 0
	}
}

// A Namespace assigns Keys to Symbols. Symbols are the literal symbols
// encountered by the lexer. Keys are handles for Symbols that contain the
// minimum information for sorting, hashing, and unification. Keys from separate
// Namespaces cannot be compared.
//
// Numeric types are keyed implicitly, and their values can be accessed and
// mutated directly from the Key. In most cases, this prevents the need to
// retrieve Numbers from the namespace.
type Namespace struct {
	// The ID is used to prevent Keys from being used with the wrong Namespace.
	// The NSID of a Key is equal to the ID of the Namespace which created it.
	// If a Namespace encounters an NSID other than its own, it panics.
	ID   int64
	gen  int
	heap *treap
}

// Name ensures that the Symbol is in the namespace, and returns its Key.
func (ns *Namespace) Name(val lex.Symbol) Key {
	switch val := val.(type) {
	case *lex.Number:
		if val.IsInt() {
			return Key{
				NSID: ns.ID,
				Type: types.Int,
				Addr: 0,
				Hash: val.Int64(),
			}
		}
		return Key{
			NSID: ns.ID,
			Type: types.Float,
			Addr: val.Float64(),
			Hash: val.Hash(),
		}

	case *lex.Functor:
		var addr float64
		addr, ns.heap = ns.heap.address(val)
		return Key{
			NSID: ns.ID,
			Type: types.Funct,
			Addr: addr,
			Hash: val.Hash(),
		}

	case *lex.Variable:
		var addr float64
		addr, ns.heap = ns.heap.address(val)
		return Key{
			NSID: ns.ID,
			Type: types.Var,
			Addr: addr,
			Hash: val.Hash(),
		}

	default:
		panic(fmt.Errorf("cannot name symbol: %v", val))
	}
}

// Value retrieves the named Symbol from the namespace.
func (ns *Namespace) Value(k Key) lex.Symbol {
	if k.NSID != ns.ID {
		panic("name used with wrong namespace (NSID mismatch)")
	}
	switch k.Type {
	case types.Int:
		val := new(lex.Number)
		val.SetInt64(k.Hash)
		return val
	case types.Float:
		val := new(lex.Number)
		val.SetFloat64(k.Addr)
		return val
	case types.Funct:
		return ns.heap.get(k.Addr)
	case types.Var:
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
	lex.Symbol
	addr        float64
	priority    int64
	lo, hi      float64
	left, right *treap
}

// get retrieves a symbol from the treap, given its address.
func (t *treap) get(addr float64) lex.Symbol {
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
func (t *treap) address(val lex.Symbol) (addr float64, root *treap) {
	if t == nil {
		return 1, &treap{
			Symbol:   val,
			addr:     1,
			priority: rand.Int63(),
			lo:       0,
			hi:       2,
		}
	}

	switch val.Cmp(t.Symbol) {
	case 0:
		return t.addr, t

	case -1:
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

	case +1:
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
	}

	return addr, root
}

func (t *treap) String() string {
	if t == nil {
		return "_"
	}
	return fmt.Sprintf("(%v.%v %v %v)", t.Symbol, t.priority, t.left, t.right)
}
