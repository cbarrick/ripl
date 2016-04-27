package scope_test

import (
	"testing"

	"github.com/cbarrick/ripl/lang/lex"
	"github.com/cbarrick/ripl/lang/scope"
)

func TestEquality(t *testing.T) {
	ns := scope.Namespace{}
	foo := lex.Functor("foo")
	foo2 := lex.Functor("foo")
	n1 := ns.Name(foo)
	n2 := ns.Name(foo2)
	if !(n1 == n2 && n1.Cmp(n2) == 0) {
		t.Error("the same name should be assigned to equal symbols")
	}
}

func TestOrder(t *testing.T) {
	ns := scope.Namespace{}

	v := lex.Variable("_1")
	num := lex.NewNumber("1")
	funct := lex.Functor("1")
	n1 := ns.Name(v)
	n2 := ns.Name(num)
	n3 := ns.Name(funct)
	if !(n1.Cmp(n2) < 0 && v.Cmp(num) < 0) {
		t.Error("variables should sort before numbers")
	}
	if !(n2.Cmp(n3) < 0 && num.Cmp(funct) < 0) {
		t.Error("numbers should sort before functors")
	}
	if !(n1.Cmp(n3) < 0 && v.Cmp(funct) < 0) {
		t.Error("variables should sort before functors")
	}

	foo := lex.Functor("foo")
	bar := lex.Functor("bar")
	n4 := ns.Name(foo)
	n5 := ns.Name(bar)
	if n4.Cmp(n5) != +1 || foo.Cmp(bar) != +1 {
		t.Error("'foo' should sort after 'bar'")
	}

	one := lex.NewNumber("1")
	two := lex.NewNumber("2")
	n6 := ns.Name(one)
	n7 := ns.Name(two)
	if !(n6.Cmp(n7) < 0 && one.Cmp(two) < 0) {
		t.Log(n6)
		t.Log(n7)
		t.Log(n6.Cmp(n7))
		t.Error("1 should sort before 2")
	}
}
