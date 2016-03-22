package term_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cbarrick/ripl/lang/term"
)

func BenchmarkLex(b *testing.B) {
	const input = "a + foo(bar,Baz) * 3.14e30 - d."
	for i := 0; i < b.N; i++ {
		for _ = range term.Lex(strings.NewReader(input)) {
		}
	}
}

func ExampleLex() {
	const input = "a + b * c - d."
	for l := range term.Lex(strings.NewReader(input)) {
		fmt.Println(&l)
	}
	// Output:
	// "a" (Functor)
	// " " (Whitespace)
	// "+" (Functor)
	// " " (Whitespace)
	// "b" (Functor)
	// " " (Whitespace)
	// "*" (Functor)
	// " " (Whitespace)
	// "c" (Functor)
	// " " (Whitespace)
	// "-" (Functor)
	// " " (Whitespace)
	// "d" (Functor)
	// "." (Terminal)
}

func ExampleLex_eof() {
	// Terminal Lexemes are inserted at EOF
	const input = "foo(bar)"
	for l := range term.Lex(strings.NewReader(input)) {
		fmt.Println(&l)
	}
	// Output:
	// "foo" (Functor)
	// "(" (Paren)
	// "bar" (Functor)
	// ")" (Paren)
	// "" (Terminal)
}
