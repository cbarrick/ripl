package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// symbols gives the valid runes for a symbol functor.
var symbols = []*unicode.RangeTable{
	unicode.Symbol,
	unicode.Pc, // punctuation, connector (contains '_')
	unicode.Pd, // punctuation, dash
	unicode.Po, // punctuation, other (contains '!', and ',')
}

// letters gives the valid runes for a letter and number functor.
var letters = []*unicode.RangeTable{
	unicode.Letter,
	unicode.Number,
	unicode.Pc, // punctuation, connector (contains '_')
}

// Scanner implements the Prolog scanner.
// It is a bufio.SplitFunc to be used with bufio.Scanner.
func Scanner(data []byte, atEOF bool) (size int, tok []byte, err error) {
	if !utf8.FullRune(data) {
		if atEOF && len(data) > 0 {
			return 0, nil, ErrBadEncoding
		}
		return 0, nil, nil
	}

	r, _ := utf8.DecodeRune(data)
	if r == utf8.RuneError {
		return 0, nil, ErrBadEncoding
	}

	switch {
	case strings.ContainsRune("!,.([{}])", r):
		return 1, data[0:1], nil
	case unicode.IsSpace(r):
		return scanSpace(data, atEOF)
	case r == '%':
		return scanComment(data, atEOF)

	// '-' could indicate number or symbol
	case r == '-':
		if len(data) < 2 || !utf8.FullRune(data[1:]) {
			if atEOF {
				return 1, data[:1], nil
			}
			return 0, nil, nil
		}
		r, _ = utf8.DecodeRune(data[1:])
		if unicode.IsNumber(r) {
			return scanNumber(data, atEOF)
		}
		return scanSymbols(data, atEOF)

	case '0' < r && r < '9':
		return scanNumber(data, atEOF)
	case r == '\'':
		return scanQuote(data, atEOF)
	case unicode.In(r, letters...):
		return scanLetters(data, atEOF)
	case unicode.In(r, symbols...):
		return scanSymbols(data, atEOF)
	default:
		return 0, nil, ErrBadEncoding
	}
}

func scanNumber(data []byte, atEOF bool) (size int, tok []byte, err error) {
	var numsize int
	l := len(data)

	if data[0] == '-' {
		size = 1
	}

	numsize, _, err = scanNatNum(data[size:], atEOF)
	if err != nil {
		return 0, nil, err
	}
	if numsize == 0 {
		return 0, nil, nil
	}
	size += numsize
	if l <= size {
		if atEOF {
			return size, data, nil
		}
		return 0, nil, nil
	}

	if data[size] == '.' {
		numsize, _, err = scanNatNum(data[size+1:], atEOF)
		if err != nil {
			return 0, nil, err
		}
		if numsize == 0 {
			return size, data[:size], nil
		}
		size += numsize + 1
	}
	if l <= size {
		if atEOF {
			return size, data, nil
		}
		return 0, nil, nil
	}

	if data[size] == 'e' {
		if size+1 < l && data[size+1] == '-' {
			numsize, _, err = scanNatNum(data[size+2:], atEOF)
		} else {
			numsize, _, err = scanNatNum(data[size+1:], atEOF)
		}
		if err != nil {
			return 0, nil, err
		}
		if numsize == 0 {
			return size, data[:size], nil
		}
		size += numsize + 1
	}
	if l <= size {
		if atEOF {
			return size, data, nil
		}
		return 0, nil, nil
	}

	return size, data[:size], nil
}

func scanNatNum(data []byte, atEOF bool) (size int, tok []byte, err error) {
	l := len(data)
	r, width := utf8.DecodeRune(data[size:])
	for size < l && unicode.IsNumber(r) {
		size += width
		r, width = utf8.DecodeRune(data[size:])
	}
	if l <= size || r == utf8.RuneError {
		if atEOF {
			return size, data[:size], nil
		}
		return 0, nil, nil
	}
	return size, data[:size], err
}

func scanLetters(data []byte, atEOF bool) (size int, tok []byte, err error) {
	l := len(data)
	r, width := utf8.DecodeRune(data)
	for size < l && unicode.In(r, letters...) {
		size += width
		r, width = utf8.DecodeRune(data[size:])
	}
	if l <= size || r == utf8.RuneError {
		return 0, nil, nil
	}
	return size, data[:size], nil
}

func scanSymbols(data []byte, atEOF bool) (size int, tok []byte, err error) {
	l := len(data)
	r, width := utf8.DecodeRune(data)
	for size < l && unicode.In(r, symbols...) {
		size += width
		r, width = utf8.DecodeRune(data[size:])
	}
	if l <= size || r == utf8.RuneError {
		return 0, nil, nil
	}
	return size, data[:size], nil
}

func scanSpace(data []byte, atEOF bool) (size int, tok []byte, err error) {
	l := len(data)
	r, width := utf8.DecodeRune(data)
	for size < l && unicode.IsSpace(r) {
		size += width
		r, width = utf8.DecodeRune(data[size:])
	}
	if l <= size || r == utf8.RuneError {
		return 0, nil, nil
	}
	return size, data[:size], nil
}

func scanComment(data []byte, atEOF bool) (size int, tok []byte, err error) {
	l := len(data)
	for size < l && data[size] != '\n' {
		size++
	}
	if size == l {
		if atEOF {
			return size, data, nil
		}
		return 0, nil, nil
	} else if data[size] == '\n' {
		size++ // count the trailing '\n' as part of the comment
	}
	return size, data[:size], nil
}

func scanQuote(data []byte, atEOF bool) (size int, tok []byte, err error) {
	l := len(data)
	if l < 2 {
		if atEOF {
			return 0, nil, ErrUnclosedQuote
		}
		return 0, nil, nil
	}

	// We know the first byte is some kind of quote.
	size = 1
	quote := data[0]

	for size < l && data[size] != quote {
		if data[size] == '\\' {
			size++
		}
		size++
	}
	size++ // count the trailing quote

	if l <= size && data[l-1] != quote {
		if atEOF {
			return 0, nil, ErrUnclosedQuote
		}
		return 0, nil, nil
	}

	return size, data[:size], nil
}
