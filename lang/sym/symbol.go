package sym

import "fmt"

// PLType identifies the Prolog type of a Symbol.
type PLType int

// Prolog types, enumerated in reverse sort order.
const (
	Funct PLType = iota
	Int
	Float
	Var
)

// Symbol is the common interface for all lexical symbols.
type Symbol interface {
	Type() PLType
	Hash() int64
	Cmp(s Symbol) int
	String() string
	Scan(state fmt.ScanState, verb rune) error
}
