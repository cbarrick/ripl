package lex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// A Lexeme is a lexical item of Prolog.
type Lexeme struct {
	Symbol
	Type
	Tok  string
	Line int
	Col  int
}

func (tok Lexeme) String() string {
	return fmt.Sprintf("%q (%v)", tok.Tok, tok.Type)
}

// A Type classifies types of lexeme.
type Type int

// The types of lexeme.
const (
	LexErr Type = iota
	SpaceTok
	CommentTok
	FunctTok
	StringTok
	NumTok
	VarTok
	ParenOpen
	ParenClose
	BracketOpen
	BracketClose
	BraceOpen
	BraceClose
	TerminalTok
)

func (typ Type) String() string {
	switch typ {
	case LexErr:
		return "Lex Error"
	case SpaceTok:
		return "Whitespace"
	case CommentTok:
		return "Comment"
	case FunctTok:
		return "Functor"
	case StringTok:
		return "String"
	case NumTok:
		return "Number"
	case VarTok:
		return "Variable"
	case ParenOpen, ParenClose,
		BracketOpen, BracketClose,
		BraceOpen, BraceClose:
		return "Paren"
	case TerminalTok:
		return "Terminal"
	default:
		panic("unknown lexeme type")
	}
}

// Norm is the form to which unicode input is normalized.
//
// SECURITY: In the NFC and NFD forms, there are look-alike characters  (e.g.
// 'K' ("\u004B") and 'K' (Kelvin sign "\u212A")). The compatibility normal
// forms, NFKC and NFKD, will map many visually nearly identical forms to a
// single value. Note that it will not do so when two symbols look alike, but
// are really from two different alphabets. For example the Latin 'o', Greek 'ο',
// and Cyrillic 'о' are still different characters as defined by these forms.
// See https://blog.golang.org/normalization.
//
// REVIEW: Is normalization a security concern at this level?
// What are the performance characteristics of each normal form?
const Norm = norm.NFC

// Lex returns all of the tokens of the next clause.
func Lex(r io.Reader) <-chan Lexeme {
	ch := make(chan Lexeme, 4)
	go lex(r, ch)
	return ch
}

// lex is the entry point of the lexing goroutine.
func lex(r io.Reader, ret chan<- Lexeme) {
	var depth int
	line, col := 1, 1
	sc := bufio.NewScanner(Norm.Reader(r))
	buf := make([]byte, 256)
	sc.Buffer(buf, bufio.MaxScanTokenSize)
	sc.Split(Scanner)

	for sc.Scan() {
		var sym Symbol
		tok := sc.Bytes()

		switch tok[0] {
		case '(', '[', '{':
			depth++
		case '}', ']', ')':
			depth--
		}

		typ := identify(tok, depth)

		// compute the change in line/col over this token
		var dl, dc int
		for _, r := range tok {
			if r == '\n' {
				dl++
				dc = -col + 1
			} else {
				dc++
			}
		}

		switch typ {
		case FunctTok:
			if tok[0] == '\'' {
				var ok bool
				tok, ok = unquote(tok)
				if !ok {
					sym = nil
					typ = LexErr
				}
			} else {
				sym = Functor(tok)
			}
		case StringTok:
			panic("strings not implemented")
		case NumTok:
			sym = NewNumber(string(tok))
		case VarTok:
			sym = Variable(tok)
		default:
			sym = nil
		}

		ret <- Lexeme{sym, typ, string(tok), line, col}

		line += dl
		col += dc
	}

	if err := sc.Err(); err != nil {
		ret <- Lexeme{nil, LexErr, err.Error(), line, col}
	}

	close(ret)
}

func identify(tok []byte, depth int) Type {
	first, _ := utf8.DecodeRune(tok)
	switch {
	case first == '.' && len(tok) == 1 && depth == 0:
		return TerminalTok
	case unicode.IsSpace(first):
		return SpaceTok
	case first == '%':
		return CommentTok
	case first == '(':
		return ParenOpen
	case first == ')':
		return ParenClose
	case first == '[':
		return BracketOpen
	case first == ']':
		return BracketClose
	case first == '{':
		return BraceOpen
	case first == '}':
		return BraceClose
	case first == '-':
		second, _ := utf8.DecodeRune(tok[1:])
		if unicode.IsNumber(second) {
			return NumTok
		}
		return FunctTok
	case unicode.IsNumber(first):
		return NumTok
	case first == '_' || unicode.IsUpper(first):
		return VarTok
	case first == '\'' || unicode.In(first, letters...) || unicode.In(first, symbols...):
		return FunctTok
	default:
		return LexErr
	}
}

// unquote removes the quotes around the token and replaces escape
// sequences. The token is modified in place, and the new slice
// is returned. If unquoting fails, the new token will contain
// an error message.
func unquote(tok []byte) ([]byte, bool) {
	// We reuse the token as the buffer.
	// This is ok: the read index will be after the write index.
	buf := bytes.NewBuffer(tok)

	// We know the first byte is a quote
	// so we start the read index at 1.
	i := 1

	// Likewise, we know the last byte is a quote
	// so we don't process it.
	end := len(tok) - 1

	for i < end {
		if tok[i] == '\\' {
			r, skip, ok := unescapeRune(tok[i:])
			if !ok {
				tok = []byte("invalid escape sequence")
				return tok, false
			}
			buf.WriteRune(r)
			i += skip
		} else {
			buf.WriteByte(tok[i])
			i++
		}
	}

	return buf.Bytes(), true
}

// unescapeRune returns the rune given by the escape sequence at
// the front of data.
func unescapeRune(data []byte) (r rune, skip int, ok bool) {
	if data[0] != '\\' {
		panic("nothing to unescape")
	}

	if len(data) < 2 {
		return 0, 0, false
	}

	switch data[1] {
	// basic escapes
	case 'a':
		return '\u0007', 2, true
	case 'b':
		return '\u0008', 2, true
	case 't':
		return '\u0009', 2, true
	case 'n':
		return '\u000A', 2, true
	case 'v':
		return '\u000B', 2, true
	case 'f':
		return '\u000C', 2, true
	case 'r':
		return '\u000D', 2, true
	case '"':
		return '\u0022', 2, true
	case '\'':
		return '\u0027', 2, true
	case '\\':
		return '\u005C', 2, true

	// octal bytes
	case '0', '1', '2', '3', '4', '5', '6', '7':
		if len(data) < 4 {
			return 0, 0, false
		}
		r |= rune(data[1]-'0') << 6
		if '0' <= data[2] && data[2] <= '7' {
			r |= rune(data[2]-'0') << 3
		} else {
			return 0, 0, false
		}
		if '0' <= data[3] && data[3] <= '7' {
			r |= rune(data[3] - '0')
		} else {
			return 0, 0, false
		}
		return r, 4, true

	// hex bytes
	case 'x':
		if len(data) < 4 {
			return 0, 0, false
		}
		d2 := unhex(data[2])
		d3 := unhex(data[3])
		r = (d2 << 4) | d3
		if d2 == utf8.RuneError || d3 == utf8.RuneError {
			return 0, 0, false
		}
		return r, 4, true

	// little code points
	case 'u':
		if len(data) < 6 {
			return 0, 0, false
		}
		d2 := unhex(data[2])
		d3 := unhex(data[3])
		d4 := unhex(data[4])
		d5 := unhex(data[5])
		r = (d2 << 12) | (d3 << 8) | (d4 << 4) | d5
		if d2 == utf8.RuneError || d3 == utf8.RuneError ||
			d4 == utf8.RuneError || d5 == utf8.RuneError {
			return 0, 0, false
		}
		return r, 6, true

	// big code points
	case 'U':
		if len(data) < 10 {
			return 0, 0, false
		}
		d2 := unhex(data[2])
		d3 := unhex(data[3])
		d4 := unhex(data[4])
		d5 := unhex(data[5])
		d6 := unhex(data[6])
		d7 := unhex(data[7])
		d8 := unhex(data[8])
		d9 := unhex(data[9])
		r = (d2 << 28) | (d3 << 24) | (d4 << 20) | (d5 << 16) |
			(d6 << 12) | (d7 << 8) | (d8 << 4) | d9
		if d2 == utf8.RuneError || d3 == utf8.RuneError ||
			d4 == utf8.RuneError || d5 == utf8.RuneError ||
			d6 == utf8.RuneError || d7 == utf8.RuneError ||
			d8 == utf8.RuneError || d9 == utf8.RuneError {
			return 0, 0, false
		}
		return r, 10, true
	}

	return 0, 0, false
}

func unhex(b byte) (r rune) {
	switch {
	case '0' <= b && b <= '9':
		return rune(b - '0')
	case 'A' <= b && b <= 'F':
		return rune(b - 'A')
	case 'a' <= b && b <= 'f':
		return rune(b - 'a')
	}
	return utf8.RuneError
}
