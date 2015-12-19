package parse

import "sort"

// An OpTable is a collection of operators for the parser. It is implemented as
// a specially sorted slice of operators. The user should not modify the table
// directly; instead use the supplied methods.
type OpTable []Op

// DefaultOps returns a new operator table extending the default table.
func DefaultOps() OpTable {
	return defaultOps[:]
}

// Get returns all operators with the given name.
func (t *OpTable) Get(name string) (s OpTable) {
	n := len(*t)
	i := t.search(name)
	j := i
	for j < n && (*t)[j].Name == name {
		j++
	}
	return (*t)[i:j]
}

// Insert puts a new operator into the table. If an operator of the same name
// and similar type (infix, prefix, or postfix) exists, it is updated instead.
func (t *OpTable) Insert(op Op) (exists bool) {
	n := len(*t)
	i := t.search(op.Name)
	j := i
	for j < n && (*t)[j].Name == op.Name {
		if ((*t)[j].Typ.infix() && op.Typ.infix()) ||
			((*t)[j].Typ.prefix() && op.Typ.prefix()) ||
			((*t)[j].Typ.postfix() && op.Typ.postfix()) {
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
	sort.Sort((*t)[i:j])
	return exists
}

// Delete removes an operator from the table.
func (t *OpTable) Delete(op Op) (exists bool) {
	n := len(*t)
	i := sort.Search(len(*t), func(i int) bool { return (*t)[i] == op })
	if i == n {
		return false
	}
	copy((*t)[i:], (*t)[i+1:])
	*t = (*t)[:n-1]
	return true
}

// Len returns the number of operators in the table.
func (t OpTable) Len() int {
	return len(t)
}

// Less reports whether the operator at index i sorts before the operator at
// index j. Operators are sorted first by descending length of name, then
// lexicographically by name, then by descending precedence, then by type.
func (t OpTable) Less(i, j int) bool {
	if len(t[i].Name) == len(t[j].Name) {
		if t[i].Name == t[j].Name {
			if t[i].Prec == t[j].Prec {
				return t[i].Typ < t[j].Typ
			}
			return t[i].Prec > t[j].Prec
		}
		return t[i].Name < t[j].Name
	}
	return len(t[i].Name) > len(t[j].Name)
}

// Swap swaps the oparators at indexes i and j.
func (t OpTable) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// search returns the first index in t such that an operator of the given name
// could appear.
func (t *OpTable) search(name string) int {
	return sort.Search(len((*t)), func(i int) bool {
		if len((*t)[i].Name) == len(name) {
			return (*t)[i].Name >= name
		}
		return len((*t)[i].Name) < len(name)
	})
}
