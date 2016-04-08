package sym

import (
	"fmt"
	"math/big"
)

// A Number represents a number in Prolog.
type Number struct {
	big.Rat
}

// NewNumber returns a pointer to a number with the value given as a string.
func NewNumber(str string) (n *Number) {
	n = new(Number)
	n.Rat.SetString(str)
	return n
}

// Type returns either Int or Float.
func (n *Number) Type() PLType {
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
	f, _ = n.Rat.Float64()
	return f
}

// Cmp compares a Number with another symbol. Numbers are sorted by value.
// Variables sort before Numbers, and everything else sorts after Numbers.
func (n *Number) Cmp(s Symbol) int {
	switch s := s.(type) {
	case *Number:
		return n.Rat.Cmp(&s.Rat)
	default:
		// PLTypes are enumerated in reverse sort order.
		return int(s.Type() - n.Type())
	}
}
