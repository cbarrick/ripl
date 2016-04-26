package lex_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cbarrick/ripl/lang/lex"
)

func BenchmarkLex(b *testing.B) {
	var input = strings.NewReader("a + foo(bar,Baz) * 3.14e30 - d.")
	for n := 0; n < b.N; n++ {
		input.Seek(0, 0)
		for _ = range lex.Lex(input) {
		}
	}
}

func ExampleLex() {
	const input = "a + b * c - d."
	for l := range lex.Lex(strings.NewReader(input)) {
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
