package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleParser_Parse() {
	var p lang.Parser
	var input = strings.NewReader("a + b * c - d.")
	p.Parse(input)
	c := p.Next()
	if p.Errs != nil {
		panic("don't panic irl")
	}
	fmt.Println("Term:", p.Canonical(c))
	// Output:
	// Term: -(+(a,*(b,c)),d)
}
