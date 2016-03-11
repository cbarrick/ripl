package lang

import "sort"

// Operators
// --------------------------------------------------

// An Op describes the parsing rules for an operator.
type Op struct {
	Prec uint   // precedence
	Typ  OpType // position and associativity
	Name string // text representation of the operator
}

// An OpType classifies types of operators.
// The zero-value is invalid; OpTypes must be initialized.
type OpType int

// The types of operators
const (
	_ OpType = iota

	FY  // associative prefix
	FX  // non-associative prefix
	XFY // left associative infix
	YFX // right associative infix
	XFX // non-associative infix
	YF  // associative postfix
	XF  // non-associative postfix
)

func (typ OpType) String() string {
	switch typ {
	case FY:
		return "fy"
	case FX:
		return "fx"
	case XFY:
		return "xfy"
	case YFX:
		return "yfx"
	case XFX:
		return "xfx"
	case YF:
		return "yf"
	case XF:
		return "xf"
	default:
		panic("unknown operator type")
	}
}

// Prefix returns true for prefix OpTypes.
func (typ OpType) Prefix() bool {
	return typ == FY || typ == FX
}

// Infix returns true for infix OpTypes.
func (typ OpType) Infix() bool {
	return typ == XFY || typ == YFX || typ == XFX
}

// Postfix returns true for postfix OpTypes.
func (typ OpType) Postfix() bool {
	return typ == XF || typ == YF
}

// Operator Table
// --------------------------------------------------

// An OpTable is a collection of operators for the parser.
type OpTable struct {
	ops []Op
}

// DefaultOps returns a new operator table extending the default table.
func DefaultOps() OpTable {
	return OpTable{defaultOps[:]}
}

// Get returns a channel yielding all operators with the given name.
func (t *OpTable) Get(name string) <-chan Op {
	ch := make(chan Op, 3) // at most 3 Ops with the same name
	for i := t.search(name); t.ops[i].Name == name; i++ {
		ch <- t.ops[i]
	}
	close(ch)
	return ch
}

// Insert puts a new operator into the table. If an operator of the same name
// and similar type (infix, prefix, or postfix) exists, it is updated instead.
func (t *OpTable) Insert(op Op) (exists bool) {
	n := len(t.ops)
	i := t.search(op.Name)
	j := i
	for j < n && t.ops[j].Name == op.Name {
		if (t.ops[j].Typ.Infix() && op.Typ.Infix()) ||
			(t.ops[j].Typ.Prefix() && op.Typ.Prefix()) ||
			(t.ops[j].Typ.Postfix() && op.Typ.Postfix()) {
			t.ops[j] = op
			exists = true
		}
		j++
	}
	if !exists {
		t.ops = append(t.ops, Op{})
		copy(t.ops[j+1:n+1], t.ops[j:n])
		t.ops[j] = op
		j++
	}
	sort.Sort(opOrd(t.ops[i:j]))
	return exists
}

// Delete removes an operator from the table.
func (t *OpTable) Delete(op Op) (exists bool) {
	n := len(t.ops)
	i := sort.Search(len(t.ops), func(i int) bool { return t.ops[i] == op })
	if i == n {
		return false
	}
	copy(t.ops[i:], t.ops[i+1:])
	t.ops = t.ops[:n-1]
	return true
}

// search returns the first index in t such that an operator of the given name
// could appear. Operators of the same name must appear consecutively.
func (t *OpTable) search(name string) int {
	return sort.Search(len(t.ops), func(i int) bool {
		if len(t.ops[i].Name) == len(name) {
			return t.ops[i].Name >= name
		}
		return len(t.ops[i].Name) < len(name)
	})
}

// The opOrd type implements operator sorting. Operators are sorted first by
// descending length of name, then lexicographically by name, then by descending
// precedence, then by type.
type opOrd []Op

func (t opOrd) Len() int {
	return len(t)
}

func (t opOrd) Less(i, j int) bool {
	if len(t[i].Name) == len(t[j].Name) {
		if t[i].Name == t[j].Name {
			if t[i].Prec == t[j].Prec {
				return t[i].Typ < t[j].Typ
			}
			return t[i].Prec > t[j].Prec
		}
		return t[i].Name < t[j].Name
	}
	return len(t[i].Name) > len(t[j].Name)
}

func (t opOrd) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
