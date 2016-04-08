package sym

import "fmt"

// PLType identifies the Prolog type of a Symbol.
type PLType int

// Prolog types.
const (
	Var PLType = iota
	Float
	Int
	Funct
)

// Symbol is the common interface for all lexical symbols.
type Symbol interface {
	String() string
	Scan(state fmt.ScanState, verb rune) error
	Hash() int64
	Cmp(s Symbol) int
}
