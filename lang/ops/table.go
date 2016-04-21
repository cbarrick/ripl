package ops

import "sort"

// An Table is a collection of operators for the parser.
type Table []Op

// Default sets the table to the default set.
func (t *Table) Default() {
	*t = Table(defaultOps[:])
}

// Get returns a channel yielding all operators with the given name.
func (t *Table) Get(name string) <-chan Op {
	ch := make(chan Op, 3) // at most 3 operators with the same name
	for i := t.search(name); (*t)[i].Name == name; i++ {
		ch <- (*t)[i]
	}
	close(ch)
	return ch
}

// Insert puts a new operator into the table. If an operator of the same name
// and similar type (infix, prefix, or postfix) exists, it is updated instead.
func (t *Table) Insert(op Op) (exists bool) {
	n := len(*t)
	i := t.search(op.Name)
	j := i
	for j < n && (*t)[j].Name == op.Name {
		if ((*t)[j].Type.Infix() && op.Type.Infix()) ||
			((*t)[j].Type.Prefix() && op.Type.Prefix()) ||
			((*t)[j].Type.Postfix() && op.Type.Postfix()) {
			(*t)[j] = op
			exists = true
		}
		j++
	}
	if !exists {
		*t = append(*t, Op{})
		copy((*t)[j+1:n+1], (*t)[j:n])
		(*t)[j] = op
		j++
	}
	sort.Sort(opOrd((*t)[i:j]))
	return exists
}

// Delete removes an operator from the table.
func (t *Table) Delete(op Op) (exists bool) {
	n := len(*t)
	i := sort.Search(n, func(i int) bool { return (*t)[i] == op })
	if i == n {
		return false
	}
	copy((*t)[i:], (*t)[i+1:])
	*t = (*t)[:n-1]
	return true
}

// search returns the first index in t such that an operator of the given name
// could appear. Operators of the same name must appear consecutively.
func (t *Table) search(name string) int {
	return sort.Search(len(*t), func(i int) bool {
		if len((*t)[i].Name) == len(name) {
			return (*t)[i].Name >= name
		}
		return len((*t)[i].Name) < len(name)
	})
}

// The opOrd type implements operator sorting. Operators are sorted first by
// descending length of name, then lexicographically by name, then by descending
// precedence, then by type.
type opOrd []Op

func (t opOrd) Len() int {
	return len(t)
}

func (t opOrd) Less(i, j int) bool {
	if len(t[i].Name) == len(t[j].Name) {
		if t[i].Name == t[j].Name {
			if t[i].Prec == t[j].Prec {
				return t[i].Type < t[j].Type
			}
			return t[i].Prec > t[j].Prec
		}
		return t[i].Name < t[j].Name
	}
	return len(t[i].Name) > len(t[j].Name)
}

func (t opOrd) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
