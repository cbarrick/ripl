package types

import "fmt"

// Value is the common interface for all Prolog values.
// Values are stored in a Namespace.
type Value interface {
	Type() ValueType
	String() string
	Scan(state fmt.ScanState, verb rune) error
}

// ValueType identifies a type of value (functor, number, etc).
type ValueType int

// Types of term.
const (
	FunctorTyp ValueType = iota
	NumberTyp
	VariableTyp
)
