package lex

import "github.com/cbarrick/ripl/lang/types"

// Symbol is the common interface for all lexical symbols.
type Symbol interface {
	Type() types.PLType
	Hash() int64
	Cmp(s Symbol) int
	String() string
}
