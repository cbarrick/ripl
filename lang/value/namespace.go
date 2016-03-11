package value

import (
	"hash"
	"hash/fnv"
)

// A Namespace stores and provides Names for Values. A Name is generated for
// each equivalent value the first time it is encountered.
type Namespace struct {
	h hash.Hash64
}

// A Name is a wrapper around a Value. It provides enough information to perform
// unification without accessing the value.
type Name struct {
	ID  uint64
	Typ ValueType
	Val Value
	Ns  *Namespace
}

// Name provides a name to a value.
// The same Name is given to equivalent Values.
func (ns *Namespace) Name(v Value) (n Name) {
	if ns.h == nil {
		ns.h = fnv.New64a()
	}
	ns.h.Write([]byte(v.String()))
	n = Name{
		ID:  ns.h.Sum64(),
		Typ: v.Type(),
		Val: v,
		Ns:  ns,
	}
	return n
}

func (n Name) String() string {
	return n.Val.String()
}
