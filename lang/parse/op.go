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

func (op *Op) prefix() bool {
	return op.Typ == FY || op.Typ == FX
}

func (op *Op) infix() bool {
	return op.Typ == XFY || op.Typ == YFX || op.Typ == XFX
}

func (op *Op) postfix() bool {
	return op.Typ == XF || op.Typ == YF
}
