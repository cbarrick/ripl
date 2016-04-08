package sym

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"
	"unicode"
)

// A Functor represents a function symbol.
type Functor struct {
	Val string
	esc bool // true when Val contains chars like '\n'
}

// NewFunctor returns a pointer to a functor with the given value.
func NewFunctor(str string) *Functor {
	f := new(Functor)
	fmt.Sscan(str, f)
	return f
}

// Type returns Funct.
func (f *Functor) Type() PLType {
	return Funct
}

// String returns the canonical representation of the functor.
func (f *Functor) String() string {
	return fmt.Sprintf("'%s'", f.Val)
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
		return f.scanBare(state)
	}
}

// scanBare scans a bare (unquoted) functor.
func (f *Functor) scanBare(state fmt.ScanState) error {
	var tok []byte
	r, _, err := state.ReadRune()
	if err != nil {
		return err
	}
	state.UnreadRune()
	switch {
	case unicode.IsLower(r):
		tok, err = state.Token(false, func(r rune) bool {
			return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
		})
		f.Val = string(tok)
		return err
	case unicode.In(r, unicode.Symbol, unicode.Pc, unicode.Pd, unicode.Po):
		tok, err = state.Token(false, func(r rune) bool {
			return unicode.In(r, unicode.Symbol, unicode.Pc, unicode.Pd, unicode.Po)
		})
		f.Val = string(tok)
		return err
	default:
		return fmt.Errorf("invalid functor")
	}
}

// scanSpecial scans functors stating with '!'.
func (f *Functor) scanSpecial(state fmt.ScanState) error {
	// The leading '!' has already been consumed from the reader.
	// Currently, the only special token we observe is the '!' cut.
	*f = Functor{
		Val: "!",
		esc: false,
	}
	return nil
}

// scanQuote scans a quoted functor.
func (f *Functor) scanQuote(state fmt.ScanState) error {
	// The leading quote has already been consumed from the reader.
	buf := new(bytes.Buffer)
	r, _, err := state.ReadRune()
	if err != nil {
		return err
	}
	f.esc = false
	for r != '\'' {
		if r == '\\' {
			f.esc = true
			r, err = scanEscape(state)
			if err != nil {
				return err
			}
			if r == '\'' {
				break
			}
		}
		buf.WriteRune(r)
		r, _, err = state.ReadRune()
		if err != nil {
			return err
		}
	}
	f.Val = buf.String()
	return nil
}

// scanEscape scans an escape character.
func scanEscape(state fmt.ScanState) (rune, error) {
	r, _, err := state.ReadRune()
	if err != nil {
		return 0, err
	}
	switch r {
	// basic escape chars
	case 'a':
		return 7, nil // ascii 'bel' (alert)
	case 'b':
		return 8, nil // ascii 'bs' (backspace)
	case 't':
		return 9, nil // ascii 'ht' (horizontal tab)
	case 'n':
		return 10, nil // ascii 'nl' (new line)
	case 'v':
		return 11, nil // ascii 'vt' (vertical tab)
	case 'f':
		return 12, nil // ascii 'np' (formfeed)
	case 'r':
		return 13, nil // ascii 'cr' (carriage return)
	case 'e':
		return 27, nil // ascii 'esc' (escape)
	case '\'':
		return '\'', nil // single quote
	case '\\':
		return '\\', nil // backslash

	// skip space
	case '\n':
		state.SkipSpace()
		r, _, err = state.ReadRune()
		if err != nil {
			return 0, err
		}
		if r == '\\' {
			return scanEscape(state)
		}
		return r, nil

	// code point escapes
	case 'x', 'u', 'U':
		var code rune
		var n uint
		switch r {
		case 'x':
			n = 2
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		for ; 0 < n; n-- {
			r, _, err = state.ReadRune()
			if err != nil {
				return 0, err
			}
			v, ok := unhex(r)
			if !ok {
				return 0, fmt.Errorf("invalid unicode escape")
			}
			code |= v << (4 * (n - 1))
		}
		return code, nil
	}

	return 0, fmt.Errorf("invalid escape")
}

func unhex(c rune) (v rune, ok bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return
}
