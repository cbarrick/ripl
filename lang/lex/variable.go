package lex

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/cbarrick/ripl/lang/types"
)

// ErrBadVar is the error returned when failing to scan a variable.
var ErrBadVar = fmt.Errorf("malformed variable")

// A Variable represents a logical variable.
type Variable string

// NewVariable returns a pointer to a variable with the given value.
func NewVariable(str string) *Variable {
	v := Variable(str)
	return &v
}

// Type returns Var.
func (v *Variable) Type() types.PLType {
	return types.Var
}

// String returns the canonical representation of the variable.
func (v *Variable) String() string {
	return string(*v)
}

// Scan scans a Variable in Prolog syntax.
func (v *Variable) Scan(state fmt.ScanState, verb rune) error {
	var tok []byte
	r, _, err := state.ReadRune()
	if err != nil {
		return err
	}
	if !unicode.IsUpper(r) && r != '_' {
		return ErrBadVar
	}
	tok, err = state.Token(false, func(r rune) bool {
		return unicode.In(r, Letters...)
	})
	*v = Variable(string(r) + string(tok))
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
		return strings.Compare(string(*v), string(*s))
	default:
		// PLTypes are enumerated in reverse sort order.
		return int(s.Type() - v.Type())
	}
}
