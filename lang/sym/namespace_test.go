package sym_test

import (
	"testing"

	"github.com/cbarrick/ripl/lang/sym"
)

func TestEquality(t *testing.T) {
	ns := new(sym.Namespace)
	foo := sym.NewFunctor("foo")
	foo2 := sym.NewFunctor("foo")
	n1 := ns.Name(foo)
	n2 := ns.Name(foo2)
	if !(n1 == n2 && n1.Cmp(n2) == 0) {
		t.Error("the same name should be assigned to equal symbols")
	}
}

func TestOrder(t *testing.T) {
	ns := new(sym.Namespace)

	v := sym.NewVariable("_1")
	num := sym.NewNumber("1")
	funct := sym.NewFunctor("1")
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

	foo := sym.NewFunctor("foo")
	bar := sym.NewFunctor("bar")
	n4 := ns.Name(foo)
	n5 := ns.Name(bar)
	if n4.Cmp(n5) != +1 || foo.Cmp(bar) != +1 {
		t.Error("'foo' should sort after 'bar'")
	}

	one := sym.NewNumber("1")
	two := sym.NewNumber("2")
	n6 := ns.Name(one)
	n7 := ns.Name(two)
	if !(n6.Cmp(n7) < 0 && one.Cmp(two) < 0) {
		t.Log(n6)
		t.Log(n7)
		t.Log(n6.Cmp(n7))
		t.Error("1 should sort before 2")
	}
}
