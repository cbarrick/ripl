package lex

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"unicode"

	"github.com/cbarrick/ripl/lang/types"
)

// ErrBadFunctor is the error returned when failing to scan a functor.
var ErrBadFunctor = fmt.Errorf("malformed functor")

// A Functor represents a function symbol.
type Functor string

// NewFunctor returns a pointer to a functor with the given value.
func NewFunctor(str string) *Functor {
	f := Functor(str)
	return &f
}

// Type returns Funct.
func (f *Functor) Type() types.PLType {
	return types.Funct
}

// String returns the canonical representation of the functor.
func (f *Functor) String() string {
	return string(*f)
}

// Hash returns the FNV-64a hash of the functor.
func (f *Functor) Hash() int64 {
	h := fnv.New64a()
	h.Write([]byte(*f))
	return int64(h.Sum64())
}

// Cmp compares a Functor with another Symbol. Functors are compared
// lexicographically. All other Symbols sort before the Functor.
func (f *Functor) Cmp(s Symbol) int {
	switch s := s.(type) {
	case *Functor:
		return strings.Compare(string(*f), string(*s))
	default:
		// PLTypes are enumerated in reverse sort order.
		return int(s.Type() - f.Type())
	}
}

// Scan scans a Functor in Prolog syntax.
func (f *Functor) Scan(state fmt.ScanState, verb rune) error {
	r, _, err := state.ReadRune()
	if err != nil {
		return err
	}
	switch r {
	case '!':
		return f.scanSpecial(state)
	case '\'':
		return f.scanQuote(state)
	default:
		state.UnreadRune()
		return f.scanBare(r, state)
	}
}

// scanBare scans a bare (unquoted) functor.
func (f *Functor) scanBare(first rune, state fmt.ScanState) (err error) {
	var tok []byte
	switch {
	case unicode.IsLower(first):
		tok, err = state.Token(false, func(r rune) bool {
			return strings.ContainsRune(ASCIILetters, r) || unicode.In(r, Letters...)
		})
		*f = Functor(tok)
		return err
	case unicode.In(first, Symbols...):
		tok, err = state.Token(false, func(r rune) bool {
			return strings.ContainsRune(ASCIISymbols, r) || unicode.In(r, Symbols...)
		})
		*f = Functor(tok)
		return err
	default:
		return ErrBadFunctor
	}
}

// scanSpecial scans functors stating with '!'.
func (f *Functor) scanSpecial(state fmt.ScanState) error {
	// The leading '!' has already been consumed from the scan state.
	// Currently, the only special token we observe is the '!' cut.
	*f = Functor("!")
	return nil
}

// scanQuote scans a quoted functor.
func (f *Functor) scanQuote(state fmt.ScanState) error {
	// The leading quote has already been consumed from the scan state.
	buf := new(bytes.Buffer)
	r, _, err := state.ReadRune()
	if err != nil {
		return err
	}

	var esc bool
	for r != '\'' {
		buf.WriteRune(r)
		if r == '\\' {
			esc = true
			r, _, err = state.ReadRune()
			if err != nil {
				return err
			}
			buf.WriteRune(r)
		}
		r, _, err = state.ReadRune()
		if err != nil {
			return err
		}
	}
	if esc {
		err = unescape(buf)
	}
	*f = Functor(buf.String())
	return err
}

// unescape replaces the contents of the buffer with the unescaped contents.
// E.g. replacing "\n" with a literal newline.
func unescape(buf *bytes.Buffer) (err error) {
	var r rune
	var s = buf.String()
	buf.Reset()
	for len(s) > 0 {
		r, _, s, err = strconv.UnquoteChar(s, '\'')
		if err != nil {
			return err
		}
		buf.WriteRune(r)
	}
	return nil
}
