package parse

type Op struct {
	Prec int    // precedence
	Typ  OpType // position and associativity
	Name string // text representation of the operator
}

type OpType int

const (
	_   OpType = iota
	FY         // associative prefix
	FX         // non-associative prefix
	XFY        // left associative infix
	YFX        // right associative infix
	XFX        // non-associative infix
	YF         // associative postfix
	XF         // non-associative postfix
)

func (typ OpType) String() string {
	switch typ {
	case FY:
		return "FY"
	case FX:
		return "FX"
	case XFY:
		return "XFY"
	case YFX:
		return "YFX"
	case XFX:
		return "XFX"
	case YF:
		return "YF"
	case XF:
		return "XF"
	default:
		return "unknown operator type"
	}
}

func (typ OpType) prefix() bool {
	return typ == FY || typ == FX
}

func (typ OpType) infix() bool {
	return typ == XFY || typ == YFX || typ == XFX
}

func (typ OpType) postfix() bool {
	return typ == XF || typ == YF
}
