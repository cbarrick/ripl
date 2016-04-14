package oper

// An Op describes the parsing rules for an operator.
type Op struct {
	Prec uint   // precedence
	Type OpType // position and associativity
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
				return t[i].Type < t[j].Type
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
