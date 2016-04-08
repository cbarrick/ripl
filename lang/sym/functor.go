package sym

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

// A Functor represents a function symbol.
type Functor struct {
	Val string
}

// NewFunctor returns a pointer to a functor with the given value.
func NewFunctor(str string) *Functor {
	return &Functor{str}
}

// String returns the canonical representation of the functor.
func (f *Functor) String() string {
	return fmt.Sprintf("'%s'", *f)
}

// Scan initializes a functor from a reader. If the first rune is a single
// quote, then the functor consists of all runes until the next single quote.
// Otherwise, the functor consists of the first token.
//
// If the atom is quoted, escape characters are processed with
// strconv.UnquoteChar.
func (f *Functor) Scan(state fmt.ScanState, verb rune) (err error) {
	var b = new(bytes.Buffer)
	var r rune
	r, _, err = state.ReadRune()
	if err != nil {
		return err
	}
	if r == '\'' {
		err = f.scanQuote(state, b)
	} else {
		var tok []byte
		b.WriteRune(r)
		if tok, err = state.Token(false, nil); err == nil {
			b.Write(tok)
		}
	}
	*f = Functor{b.String()}
	return err
}

func (f *Functor) scanQuote(state fmt.ScanState, b *bytes.Buffer) (err error) {
	var r rune
	for r != '\'' && err == nil {
		r, _, err = state.ReadRune()
		if r == '\\' {
			if r, _, err = state.ReadRune(); err == nil {
				r, _, _, err = strconv.UnquoteChar("\\"+string(r), 0)
			}
		}
		b.WriteRune(r)
	}
	return err
}

// Hash returns the FNV-64a hash of the functor.
func (f *Functor) Hash() int64 {
	h := fnv.New64a()
	h.Write([]byte(f.Val))
	return int64(h.Sum64())
}

// Cmp compares a Functor with another Symbol. Functors are compared
// lexicographically. All other Symbols sort before the Functor.
func (f *Functor) Cmp(s Symbol) int {
	switch s := s.(type) {
	case *Functor:
		return strings.Compare(f.Val, s.Val)
	default:
		return +1
	}
}
