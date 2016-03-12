package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleParse() {
	var input = strings.NewReader("a + b * c - d")
	var ops = lang.DefaultOps()
	var ns = new(lang.Namespace)
	var c = make(lang.Clause, 4)
	t, err := c.Parse(input, ops, ns)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Term:", t)
	}
	fmt.Println("Heap:", c)
	// Output:
	// Term: -(+(a,*(b,c)),d)
	// Heap: [b c a *(b,c) +(a,*(b,c)) d -(+(a,*(b,c)),d)]
}
