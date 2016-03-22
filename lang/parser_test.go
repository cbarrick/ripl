package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleParse() {
	var input = strings.NewReader("a + b * c - d")
	var ops = lang.DefaultOps()
	var ns = lang.NewNamespace(16)
	c, err := lang.Parse(input, ops, ns)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Term:", c.Root())
	}
	// Output:
	// Term: -(+(a,*(b,c)),d)
}
