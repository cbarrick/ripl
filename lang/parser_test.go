package lang_test

import (
	"fmt"
	"strings"

	"github.com/cbarrick/ripl/lang"
)

func ExampleParse() {
	var clause = "a + b * c - d"
	var t = new(lang.Term)
	var heap = make([]lang.Term, 0, 8)
	var ops = lang.DefaultOps()
	heap, err := t.Parse(strings.NewReader(clause), ops, heap)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("Term:", t)
	fmt.Println("Heap:", heap)
	// Output:
	// Term: -(+(a,*(b,c)),d)
	// Heap: [b c a *(b,c) +(a,*(b,c)) d]
}
