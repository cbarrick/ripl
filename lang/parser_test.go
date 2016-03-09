package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleParse() {
	var input = "a + b * c - d"
	var heap = make([]lang.Term, 0, 8)
	var ops = lang.DefaultOps()
	c, heap, err := lang.Parse(strings.NewReader(input), heap, ops)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("Term:", c.Root())
	fmt.Println("Heap:", heap)
	// Output:
	// Term: -(+(a,*(b,c)),d)
	// Heap: [b c a *(b,c) +(a,*(b,c)) d -(+(a,*(b,c)),d)]
}
