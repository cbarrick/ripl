package types

// PLType identifies the Prolog type of a Symbol.
type PLType int

// Prolog types, enumerated in reverse sort order.
const (
	Funct PLType = iota
	Int
	Float
	Var
)
