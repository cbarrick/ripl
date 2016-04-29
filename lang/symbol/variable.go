package symbol

import "strings"

// A Variable represents a logical variable.
type Variable string

// Type returns Var.
func (v Variable) Type() Type {
	return Var
}

// String returns the canonical representation of the variable.
func (v Variable) String() string {
	return string(v)
}

// Hash returns 0.
func (v Variable) Hash() int64 {
	return 0
}

// Cmp compares a Variable with another Symbol. Variables are compared
// lexicographically. All other Symbols sort after the Variable.
func (v Variable) Cmp(s Symbol) int {
	switch s := s.(type) {
	case Variable:
		return strings.Compare(string(v), string(s))
	default:
		// PLTypes are enumerated in reverse sort order.
		return int(s.Type() - v.Type())
	}
}
