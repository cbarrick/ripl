package value

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"unicode"
)

// Value is the common interface for all Prolog values.
// Values are stored in a Namespace.
type Value interface {
	String() string
	Scan(state fmt.ScanState, verb rune) error
	Type() ValueType
}

// ValueType identifies a type of value (functor, number, etc).
type ValueType int

// Types of term.
const (
	ErrorTyp ValueType = iota
	FunctorTyp
	VariableTyp
	StringTyp
	NumberTyp
	ListTyp
)

// A Functor is a text value.
type Functor bytes.Buffer

func (f *Functor) String() string {
	return (*bytes.Buffer)(f).String()
}

func (f *Functor) Scan(state fmt.ScanState, verb rune) (err error) {
	var b = (*bytes.Buffer)(f)
	var tok []byte
	var r rune
	r, _, err = state.ReadRune()
	if err != nil {
		return err
	}
	if r == '\'' {
		return f.scanQuote(state)
	}
	b.WriteRune(r)
	if tok, err = state.Token(false, nil); err == nil {
		b.Write(tok)
	}
	return err
}

func (f *Functor) scanQuote(state fmt.ScanState) (err error) {
	var b = (*bytes.Buffer)(f)
	var r rune
	r, _, err = state.ReadRune()
	for r != '\'' && err == nil {
		if r == '\\' {
			if r, _, err = state.ReadRune(); err == nil {
				r, _, _, err = strconv.UnquoteChar("\\"+string(r), 0)
			}
		}
		b.WriteRune(r)
		r, _, err = state.ReadRune()
	}
	return err
}

func (*Functor) Type() ValueType {
	return FunctorTyp
}

// A Number is a Prolog number.
// Ripl uses a single Number type for both floats and ints.
type Number big.Rat

func (n *Number) String() string {
	x := (*big.Rat)(n)
	if x.IsInt() {
		return x.Num().String()
	}
	f, _ := x.Float64()
	return fmt.Sprint(f)
}

func (n *Number) Scan(state fmt.ScanState, verb rune) error {
	x := (*big.Rat)(n)
	_, err := fmt.Fscan(state, x)
	return err
}

func (*Number) Type() ValueType {
	return NumberTyp
}

// A Variable is a Prolog variable.
type Variable string

func (v *Variable) String() string {
	return fmt.Sprint("_", string(*v))
}

func (v *Variable) Scan(state fmt.ScanState, verb rune) (err error) {
	var r rune
	var tok []byte
	buf := new(bytes.Buffer)
	if r, _, err = state.ReadRune(); err == nil {
		if r == '_' || unicode.IsUpper(r) {
			if _, err = buf.WriteRune(r); err == nil {
				if tok, err = state.Token(false, nil); err == nil {
					buf.Write(tok)
				}
			}
		}
	}
	*v = Variable(buf.String())
	return err
}

func (*Variable) Type() ValueType {
	return VariableTyp
}

// Error values are used for handling I/O errors.
type Error struct{ E error }

func (err Error) String() string {
	return err.E.Error()
}

func (err Error) Scan(state fmt.ScanState, verb rune) error {
	panic("cannot scan Error values")
}

func (Error) Type() ValueType {
	return ErrorTyp
}
