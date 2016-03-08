package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleLex() {
	for l := range lang.Lex(strings.NewReader("a + b * c - d.")) {
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
