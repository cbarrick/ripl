package term

import "github.com/cbarrick/ripl/lang/types"

type Namespace struct {
	names map[string]Name
}

type Name struct {
	Val types.Value
	Typ types.ValueType
}

func NewNamespace(cap int) *Namespace {
	return &Namespace{
		names: make(map[string]Name, cap),
	}
}

func (ns *Namespace) Name(v types.Value) (n Name) {
	key := v.String()
	n = ns.names[key]
	if n.Val == nil {
		n = Name{
			Val: v,
			Typ: v.Type(),
		}
		ns.names[key] = n
	}
	return n
}

func (n Name) String() string {
	return n.Val.String()
}
