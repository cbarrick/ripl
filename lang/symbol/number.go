package symbol

import (
	"fmt"
	"math/big"
)

// A Number represents a number in Prolog.
type Number big.Rat

// NewNumber returns a pointer to a number with the value given as a string.
func NewNumber(str string) (n *Number) {
	n = new(Number)
	(*big.Rat)(n).SetString(str)
	return n
}

// Type returns either Int or Float.
func (n *Number) Type() Type {
	if n.IsInt() {
		return Int
	}
	return Float
}

// String returns the canonical representation of the number.
func (n *Number) String() string {
	if n.IsInt() {
		return n.Num().String()
	}
	f := n.Float64()
	return fmt.Sprint(f)
}

// Scan scans a Number in Prolog syntax.
func (n *Number) Scan(state fmt.ScanState, verb rune) error {
	return (*big.Rat)(n).Scan(state, verb)
}

// Hash returns the integer part of the number.
func (n *Number) Hash() int64 {
	return n.Num().Int64()
}

// Int64 returns the integer part of the number.
func (n *Number) Int64() int64 {
	return n.Num().Int64()
}

// Float64 returns the number as a float64.
func (n *Number) Float64() (f float64) {
	f, _ = (*big.Rat)(n).Float64()
	return f
}

// IsInt returns true if n is an integer.
func (n *Number) IsInt() bool {
	return (*big.Rat)(n).IsInt()
}

// Num returns the integer part of the number.
func (n *Number) Num() *big.Int {
	return (*big.Rat)(n).Num()
}

// Cmp compares a Number with another symbol. Numbers are sorted by value.
// Variables sort before Numbers, and everything else sorts after Numbers.
func (n *Number) Cmp(s Symbol) int {
	switch s := s.(type) {
	case *Number:
		return (*big.Rat)(n).Cmp((*big.Rat)(s))
	default:
		return int(n.Type() - s.Type())
	}
}

// SetInt64 sets the value of the number to x.
func (n *Number) SetInt64(x int64) {
	(*big.Rat)(n).SetInt64(x)
}

// SetFloat64 sets the value of the number to x.
func (n *Number) SetFloat64(x float64) {
	(*big.Rat)(n).SetFloat64(x)
}
