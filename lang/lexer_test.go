package lang_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cbarrick/ripl/lang"
)

func TestLex(*testing.T) {
	for l := range lang.Lex(strings.NewReader("a + b * c - d. ")) {
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