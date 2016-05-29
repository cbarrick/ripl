package lexer_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cbarrick/ripl/lang/lexer"
)

func BenchmarkLex(b *testing.B) {
	var input = strings.NewReader("a + foo(bar,Baz) * 3.14e30 - d.")
	for n := 0; n < b.N; n++ {
		input.Seek(0, 0)
		for _ = range lexer.Read(input) {
		}
	}
}

func ExampleLex() {
	var input = strings.NewReader("a + foo(bar,Baz) * 3.14e30 - d.")
	for l := range lexer.Read(input) {
		fmt.Println(l)
	}
	// Output:
	// "a" (Functor)
	// " " (Whitespace)
	// "+" (Functor)
	// " " (Whitespace)
	// "foo" (Functor)
	// "(" (Paren)
	// "bar" (Functor)
	// "," (Functor)
	// "Baz" (Variable)
	// ")" (Paren)
	// " " (Whitespace)
	// "*" (Functor)
	// " " (Whitespace)
	// "3.14e30" (Number)
	// " " (Whitespace)
	// "-" (Functor)
	// " " (Whitespace)
	// "d" (Functor)
	// "." (Terminal)
}