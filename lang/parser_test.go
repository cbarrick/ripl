package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
	"github.com/cbarrick/ripl/lang/ops"
	"github.com/cbarrick/ripl/lang/scope"
)

func ExampleParse() {
	var input = strings.NewReader("a + b * c - d.")
	var optab = ops.Default()
	var ns = new(scope.Namespace)
	p := lang.Parse(input, optab, ns)
	c := p.Next()
	if p.Errs != nil {
		panic("don't panic irl")
	}
	fmt.Println("Term:", c)
	// Output:
	// Term: -(+(a,*(b,c)),d)
}
