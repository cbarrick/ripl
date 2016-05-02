package symbol

import (
	"hash/fnv"
	"strings"
)

// A Functor represents a function symbol.
type Functor string

// Type returns Funct.
func (f Functor) Type() Type {
	return Funct
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
		return int(f.Type() - s.Type())
	}
}
