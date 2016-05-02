package symbol_test

import (
	"testing"

	"github.com/cbarrick/ripl/lang/symbol"
)

func TestSimple(t *testing.T) {
	var ns symbol.Namespace
	foo := symbol.Functor("foo")
	nfoo := ns.Name(foo)
	if ns.Value(nfoo) != foo {
		t.Error("the value of a name must match the original symbol")
	}
}

func TestEquality(t *testing.T) {
	var ns symbol.Namespace
	foo := symbol.Functor("foo")
	foo2 := symbol.Functor("foo")
	n1 := ns.Name(foo)
	n2 := ns.Name(foo2)
	if !(n1 == n2 && n1.Cmp(n2) == 0) {
		t.Error("the same name should be assigned to equal symbols")
	}
}

func TestOrder(t *testing.T) {
	var ns symbol.Namespace

	v := symbol.Variable("_1")
	num := symbol.NewNumber("1")
	funct := symbol.Functor("1")
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

	foo := symbol.Functor("foo")
	bar := symbol.Functor("bar")
	n4 := ns.Name(foo)
	n5 := ns.Name(bar)
	if !(0 < n4.Cmp(n5) && 0 < foo.Cmp(bar)) {
		t.Error("'foo' should sort after 'bar'")
	}
}
