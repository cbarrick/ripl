package value

// A Namespace stores and provides Names for Values. A Name is generated for
// each equivalent value the first time it is encountered.
type Namespace struct {
	init   bool // true when initialized
	names  map[string]Name
	values []Value
}

// A Name is used to retrieve a Value from a Namespace. Equal names belonging to
// the same Namespace map to equal Values.
type Name struct {
	Typ ValueType
	id  int
	ns  *Namespace
}

// Init initializes a Namespace with capacity cap.
func (ns *Namespace) Init(cap int) {
	*ns = Namespace{
		init:   true,
		names:  make(map[string]Name, cap),
		values: make([]Value, 0, cap),
	}
}

// Name provides a name to a value. The same Name is given to equal Values.
// If a Value equal to v has never been seen, v is retained.
func (ns *Namespace) Name(v Value) (n Name) {
	if !ns.init {
		ns.Init(0)
	}
	str := v.String()
	n = ns.names[str]
	if n.id == 0 {
		n = Name{
			Typ: v.Type(),
			id:  len(ns.values) + 1,
			ns:  ns,
		}
		ns.values = append(ns.values, v)
		ns.names[str] = n
	}
	return n
}

// Value retrieves the named value.
func (n Name) Value() (v Value) {
	return n.ns.values[n.id-1]
}
