package term_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang/oper"
	"github.com/cbarrick/ripl/lang/term"
)

func ExampleParse() {
	var input = strings.NewReader("a + b * c - d")
	var ops = oper.DefaultOps()
	var ns = term.NewNamespace(16)
	c, err := term.Parse(input, ops, ns)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Term:", c.Root())
	}
	// Output:
	// Term: -(+(a,*(b,c)),d)
}
