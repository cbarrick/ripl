package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleLex() {
	var clause = "a + b * c - d."
	for l := range lang.Lex(strings.NewReader(clause)) {
		fmt.Println(&l)
	}
	// Output:
	// "a" (Funct)
	// " " (Space)
	// "+" (Funct)
	// " " (Space)
	// "b" (Funct)
	// " " (Space)
	// "*" (Funct)
	// " " (Space)
	// "c" (Funct)
	// " " (Space)
	// "-" (Funct)
	// " " (Space)
	// "d" (Funct)
	// '.' (Terminal)
}

func ExampleLex_eof() {
	// Terminal Lexemes are inserted at EOF
	var clause = "foo(bar)"
	for l := range lang.Lex(strings.NewReader(clause)) {
		fmt.Println(&l)
	}
	// Output:
	// "foo" (Funct)
	// '(' (Paren)
	// "bar" (Funct)
	// ')' (Paren)
	// '\x03' (Terminal)
}
