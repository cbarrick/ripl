package term_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang/ops"
	"github.com/cbarrick/ripl/lang/scope"
	"github.com/cbarrick/ripl/lang/term"
)

func ExampleParse() {
	var input = strings.NewReader("a + b * c - d")
	var optab = ops.Default()
	var ns = new(scope.Namespace)
	c, err := term.Parse(input, optab, ns)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Term:", c)
	}
	// Output:
	// Term: -(+(a,*(b,c)),d)
}
