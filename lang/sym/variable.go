package sym

import (
	"fmt"
	"strings"
	"unicode"
)

// A Variable represents a logical variable.
type Variable struct {
	Val string
}

// NewVariable returns a pointer to a variable with the given value.
func NewVariable(str string) *Variable {
	return &Variable{str}
}

// Type returns Var.
func (v *Variable) Type() PLType {
	return Var
}

// String returns the canonical representation of the variable.
func (v *Variable) String() string {
	return v.Val
}

// Scan initializes a variable from a reader.
// The first token is taken as the variable name.
func (v *Variable) Scan(state fmt.ScanState, verb rune) error {
	var tok []byte
	r, _, err := state.ReadRune()
	if err != nil {
		return err
	}
	if !unicode.IsUpper(r) && r != '_' {
		return fmt.Errorf("invalid variable name")
	}
	tok, err = state.Token(false, func(r rune) bool {
		return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
	})
	*v = Variable{string(r) + string(tok)}
	return err
}

// Hash returns 0.
func (v *Variable) Hash() int64 {
	return 0
}

// Cmp compares a Variable with another Symbol. Variables are compared
// lexicographically. All other Symbols sort after the Variable.
func (v *Variable) Cmp(s Symbol) int {
	switch s := s.(type) {
	case *Variable:
		return strings.Compare(v.Val, s.Val)
	default:
		// PLTypes are enumerated in reverse sort order.
		return int(s.Type() - v.Type())
	}
}
