package lang_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cbarrick/ripl/lang"
	"github.com/cbarrick/ripl/lang/operator"
	"github.com/cbarrick/ripl/lang/symbol"
)

func ExampleParser_Parse() {
	var input = strings.NewReader("a + b * c - d.")
	var ops = operator.Default()
	var ns symbol.Namespace
	var p = lang.Parser{
		OpTab:  &ops,
		SymTab: &ns,
	}
	p.Parse(input)
	c, _ := p.Next()
	if p.Errs != nil {
		panic("don't panic irl")
	}
	fmt.Println("Term:", p.Canonical(c))
	// Output:
	// Term: -(+(a,*(b,c)),d)
}

func TestParser(t *testing.T) {
	var input = strings.NewReader("foo(X) :- X = bar.")
	var ops = operator.Default()
	var ns symbol.Namespace
	var p = lang.Parser{
		OpTab:  &ops,
		SymTab: &ns,
	}
	p.Parse(input)
	c, ok := p.Next()
	if !ok {
		t.Error("parser closed early")
	}
	if p.Errs != nil {
		t.Log("parser errors:")
		for _, err := range p.Errs {
			t.Log(err.Error())
		}
		t.FailNow()
	}
	if p.Canonical(c) != ":-(foo(X),=(X,bar))" {
		t.Error("incorrect parse tree")
	}
}
