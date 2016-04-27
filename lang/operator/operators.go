package operator

// An Op describes the parsing rules for an operator.
type Op struct {
	Type        // position and associativity
	Prec uint   // precedence
	Name string // text representation of the operator
}

// An Type classifies types of operators.
// The zero-value is invalid; OpTypes must be initialized.
type Type int

// The types of operators
const (
	_ Type = iota

	FY  // associative prefix
	FX  // non-associative prefix
	XFY // left associative infix
	YFX // right associative infix
	XFX // non-associative infix
	YF  // associative postfix
	XF  // non-associative postfix
)

func (typ Type) String() string {
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
func (typ Type) Prefix() bool {
	return typ == FY || typ == FX
}

// Infix returns true for infix OpTypes.
func (typ Type) Infix() bool {
	return typ == XFY || typ == YFX || typ == XFX
}

// Postfix returns true for postfix OpTypes.
func (typ Type) Postfix() bool {
	return typ == XF || typ == YF
}
