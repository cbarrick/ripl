package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleLex() {
	var input = "a + b * c - d."
	for l := range lang.Lex(strings.NewReader(input)) {
		fmt.Println(&l)
	}
	// Output:
	// 	"a" (Functor)
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
	var input = "foo(bar)"
	for l := range lang.Lex(strings.NewReader(input)) {
		fmt.Println(&l)
	}
	// Output:
	// "foo" (Functor)
	// "(" (Paren)
	// "bar" (Functor)
	// ")" (Paren)
	// "." (Terminal)
}
