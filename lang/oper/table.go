package oper

import "sort"

// An Table is a collection of operators for the parser.
type Table struct {
	ops []Op
}

// DefaultOps returns a new operator table extending the default table.
func DefaultOps() Table {
	return Table{defaultOps[:]}
}

// Get returns a channel yielding all operators with the given name.
func (t *Table) Get(name string) <-chan Op {
	ch := make(chan Op, 3) // at most 3 Ops with the same name
	for i := t.search(name); t.ops[i].Name == name; i++ {
		ch <- t.ops[i]
	}
	close(ch)
	return ch
}

// Insert puts a new operator into the table. If an operator of the same name
// and similar type (infix, prefix, or postfix) exists, it is updated instead.
func (t *Table) Insert(op Op) (exists bool) {
	n := len(t.ops)
	i := t.search(op.Name)
	j := i
	for j < n && t.ops[j].Name == op.Name {
		if (t.ops[j].Typ.Infix() && op.Typ.Infix()) ||
			(t.ops[j].Typ.Prefix() && op.Typ.Prefix()) ||
			(t.ops[j].Typ.Postfix() && op.Typ.Postfix()) {
			t.ops[j] = op
			exists = true
		}
		j++
	}
	if !exists {
		t.ops = append(t.ops, Op{})
		copy(t.ops[j+1:n+1], t.ops[j:n])
		t.ops[j] = op
		j++
	}
	sort.Sort(opOrd(t.ops[i:j]))
	return exists
}

// Delete removes an operator from the table.
func (t *Table) Delete(op Op) (exists bool) {
	n := len(t.ops)
	i := sort.Search(len(t.ops), func(i int) bool { return t.ops[i] == op })
	if i == n {
		return false
	}
	copy(t.ops[i:], t.ops[i+1:])
	t.ops = t.ops[:n-1]
	return true
}

// search returns the first index in t such that an operator of the given name
// could appear. Operators of the same name must appear consecutively.
func (t *Table) search(name string) int {
	return sort.Search(len(t.ops), func(i int) bool {
		if len(t.ops[i].Name) == len(name) {
			return t.ops[i].Name >= name
		}
		return len(t.ops[i].Name) < len(name)
	})
}
