package sym

import (
	"bytes"
	"fmt"
	"strings"
)

// A Variable represents a logical variable.
type Variable struct {
	Val string
}

// NewVariable returns a pointer to a variable with the given value.
func NewVariable(str string) *Variable {
	return &Variable{str}
}

// String returns the canonical representation of the variable.
func (v *Variable) String() string {
	return v.Val
}

// Scan initializes a variable from a reader.
// The first token is taken as the variable name.
func (v *Variable) Scan(state fmt.ScanState, verb rune) (err error) {
	var tok []byte
	buf := new(bytes.Buffer)
	if tok, err = state.Token(false, nil); err == nil {
		buf.Write(tok)
	}
	*v = Variable{buf.String()}
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
		return -1
	}
}
