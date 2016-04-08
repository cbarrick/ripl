package sym

import "math/rand"

// A Name refers to a symbol in a Namespace. Names contain the minimal
// information to compare named symbols without accessing the symbol itself.
//
// The address of a name is used to retrieve the named symbol from the
// namespace. While the semantics of the address are well defined, the meaning
// is different for symbols of different Prolog type:
//
// For functors, the address represents a lexicographic ordering of functors in
// the same namespace. For floats, the address is the value of the float. For
// integers, the address is always 0. And for variables, the address represents
// an arbitrary total ordering of variables.
//
// The hash of a name is equal to the hash of the named symbol. For integers,
// the hash is the value of the integer. For variables, the hash is always zero.
//
// The total ordering of names can be derived from its type, address, and hash.
type Name struct {
	NSID int64
	Type PLType
	Addr float64
	Hash int64
}

// Cmp compares named symbols. It is logically equivalent to calling Cmp on the
// symbols directly. Names from different namespaces cannot be compared.
func (n Name) Cmp(m Name) int {
	switch {
	case n.NSID != m.NSID:
		panic("cannot compare names from different namespaces (NSID mismatch)")

	// PLTypes are enumerated in reverse sort order.
	case m.Type < n.Type:
		return -1
	case n.Type < m.Type:
		return +1

	case n.Type == Int:
		return int(n.Hash - m.Hash)
	case n.Addr < m.Addr:
		return -1
	case m.Addr < n.Addr:
		return +1
	default:
		return 0
	}
}

// A Namespace assigns Names to Symbols. In compilers parlance, this is our
// symbol table. Symbols are the literal symbols encountered by the lexer. Names
// are handles for Symbols that contain the minimum information for sorting,
// hashing, and unification. Names from separate namespaces cannot be compared.
//
// Numeric types are named implicitly, and their values can be accessed and
// mutated directly from the name. This prevents most needs to retrieve a Number
// symbol from the namespace.
type Namespace struct {
	// The ID is used to prevent Names from being used with the wrong Namespace.
	// The NSID of a Name is equal to the ID of the Namespace which created it.
	// If a Namespace encounters an NSID other than its own, it panics.
	ID   int64
	gen  int
	heap *treap
}

// Name ensures that the symbol is in the namespace, and returns its name.
func (ns *Namespace) Name(val Symbol) Name {
	switch val := val.(type) {
	case *Number:
		if val.IsInt() {
			return Name{
				NSID: ns.ID,
				Type: Int,
				Addr: 0,
				Hash: val.Int64(),
			}
		}
		return Name{
			NSID: ns.ID,
			Type: Float,
			Addr: val.Float64(),
			Hash: val.Hash(),
		}

	case *Functor:
		var addr float64
		addr, ns.heap = ns.heap.address(val)
		return Name{
			NSID: ns.ID,
			Type: Funct,
			Addr: addr,
			Hash: val.Hash(),
		}

	case *Variable:
		var addr float64
		addr, ns.heap = ns.heap.address(val)
		return Name{
			NSID: ns.ID,
			Type: Var,
			Addr: addr,
			Hash: val.Hash(),
		}

	default:
		panic("unknown Symbol type")
	}
}

// Value retrieves the named symbol from the namespace.
func (ns *Namespace) Value(n Name) Symbol {
	if n.NSID != ns.ID {
		panic("name used with wrong namespace (NSID mismatch)")
	}
	switch n.Type {
	case Int:
		val := new(Number)
		val.SetInt64(n.Hash)
		return val
	case Float:
		val := new(Number)
		val.SetFloat64(n.Addr)
		return val
	case Funct:
		return ns.heap.get(n.Addr)
	case Var:
		return ns.heap.get(n.Addr)
	default:
		panic("unknown Prolog type")
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
	Symbol
	addr        float64
	priority    int64
	prev, next  *treap
	left, right *treap
}

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
func (t *treap) address(val Symbol) (a float64, root *treap) {
	if t == nil {
		return 1, &treap{
			Symbol:   val,
			addr:     1,
			priority: rand.Int63(),
		}
	}

	switch val.Cmp(t.Symbol) {
	case 0:
		return t.addr, t

	case -1:
		var prev *treap
		if t.left == nil {
			root = new(treap)
			*root = *t
			a = t.genAddrBefore()
			prev = &treap{
				Symbol:   val,
				addr:     a,
				prev:     root.prev,
				next:     root,
				priority: rand.Int63(),
			}
			root.prev = prev
		} else {
			a, prev = t.left.address(val)
			if prev == t.left {
				return a, t
			}
			root = new(treap)
			*root = *t
		}
		if prev.priority > root.priority {
			prev.right, root.left = root, prev.right
			root = prev
		}

	case +1:
		var next *treap
		if t.right == nil {
			root = new(treap)
			*root = *t
			a = t.genAddrAfter()
			next = &treap{
				Symbol:   val,
				addr:     a,
				prev:     root,
				next:     root.next,
				priority: rand.Int63(),
			}
			root.next = next
		} else {
			a, next = t.right.address(val)
			if next == t.right {
				return a, t
			}
			root = new(treap)
			*root = *t
		}
		if next.priority > root.priority {
			next.left, root.right = root, next.left
			root = next
		}
	}

	return a, root
}

func (t *treap) genAddrBefore() float64 {
	if t.prev == nil {
		return t.addr / 2
	}
	return t.addr/2 + t.prev.addr/2
}

func (t *treap) genAddrAfter() float64 {
	if t.next == nil {
		return t.addr * 2
	}
	return t.addr/2 + t.next.addr/2
}
