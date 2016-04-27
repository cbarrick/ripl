package lex

import (
	"hash/fnv"
	"strings"

	"github.com/cbarrick/ripl/lang/types"
)

// A Functor represents a function symbol.
type Functor string

// Type returns Funct.
func (f Functor) Type() types.PLType {
	return types.Funct
}

// String returns the canonical representation of the functor.
func (f Functor) String() string {
	return string(f)
}

// Hash returns the FNV-64a hash of the functor.
func (f Functor) Hash() int64 {
	h := fnv.New64a()
	h.Write([]byte(f))
	return int64(h.Sum64())
}

// Cmp compares a Functor with another Symbol. Functors are compared
// lexicographically. All other Symbols sort before the Functor.
func (f Functor) Cmp(s Symbol) int {
	switch s := s.(type) {
	case Functor:
		return strings.Compare(string(f), string(s))
	default:
		// PLTypes are enumerated in reverse sort order.
		return int(s.Type() - f.Type())
	}
}
