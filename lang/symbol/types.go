package symbol

// Type identifies the Prolog type of a Symbol.
type Type int

// Prolog types, enumerated in reverse sort order.
const (
	Funct Type = iota
	Int
	Float
	Var
)
