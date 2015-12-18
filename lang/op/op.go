package op

import (
	"sort"
	"sync"
)

// Operator
// --------------------------------------------------

type Op struct {
	Prec int    // precedence
	Typ  OpType // position and associativity
	Name string // text representation of the operator
}

type OpType int

const (
	_   OpType = iota
	FY         // associative prefix
	FX         // non-associative prefix
	XFY        // left associative infix
	YFX        // right associative infix
	XFX        // non-associative infix
	YF         // associative postfix
	XF         // non-associative postfix
)

func (op *Op) prefix() bool {
	return op.Typ == FY || op.Typ == FX
}

func (op *Op) infix() bool {
	return op.Typ == XFY || op.Typ == YFX || op.Typ == XFX
}

func (op *Op) postfix() bool {
	return op.Typ == XF || op.Typ == YF
}

// Operator Table
// --------------------------------------------------

type OpTable struct {
	sync.RWMutex
	parent *OpTable
	tab    map[string][]Op
}

// DefaultOps returns a new operator table extending the default table.
func DefaultOps() OpTable {
	return ExtendOps(&defaultOps)
}

// ExtendOps returns a new operator table extending the parent table.
func ExtendOps(parent *OpTable) OpTable {
	return OpTable{
		parent: parent,
		tab:    make(map[string][]Op),
	}
}

// Get returns all operators with the given name.
func (t *OpTable) Get(name string) (s []Op) {
	t.RLock()
	defer t.RUnlock()
	ops := t.tab[name]
	s = make([]Op, len(ops))
	copy(s, ops)
	if t.parent != nil {
		s = append(s, t.parent.Get(name)...)
	}
	return s
}

// Add inserts a new operator into the table.
// Add does not effect the parent table.
func (t *OpTable) Add(op Op) (exists bool) {
	t.Lock()
	defer t.Unlock()
	s := t.tab[op.Name]
	for i := range s {
		if s[i].infix() && op.infix() ||
			s[i].prefix() && op.prefix() ||
			s[i].postfix() && op.postfix() {
			s[i] = op
			return true
		}
	}
	s = append(s, op)
	t.tab[op.Name] = s
	return false
}

// Delete removes an operator from the table.
// Delete does not effect the parent table.
func (t *OpTable) Delete(op Op) (exists bool) {
	t.Lock()
	defer t.Unlock()
	s := t.tab[op.Name]
	size := len(s)
	for i := range s {
		if s[i] == op {
			s[i], s[size-1] = s[size-1], s[i]
			s = s[:size-1]
			if size == 1 {
				delete(t.tab, op.Name)
			} else {
				t.tab[op.Name] = s
			}
			return true
		}
	}
	return false
}

// Len returns the number of operators in the table.
func (t *OpTable) Len() int {
	t.RLock()
	defer t.RUnlock()
	size := 0
	for _, v := range t.tab {
		size += len(v)
	}
	if t.parent != nil {
		size += t.parent.Len()
	}
	return size
}

// Slice returns the operators as a slice.
func (t *OpTable) Slice() (s []Op) {
	t.RLock()
	defer t.RUnlock()
	for _, v := range t.tab {
		for _, op := range v {
			s = append(s, op)
		}
	}
	if t.parent != nil {
		s = append(s, t.parent.Slice()...)
	}
	return s
}

// ByName returns the operators sorted lexicographically.
func (t *OpTable) ByName() (s []Op) {
	s = t.Slice()
	sort.Sort(ByName{s})
	return s
}

// ByShortest returns the operators sorted by longest name.
func (t *OpTable) ByLongest() (s []Op) {
	s = t.Slice()
	sort.Sort(sort.Reverse(ByLen{s}))
	return s
}

// ByPrec returns the operators sorted by descending precedence.
func (t *OpTable) ByPrec() (s []Op) {
	s = t.Slice()
	sort.Sort(ByPrec{s})
	return s
}

// Sorting
// --------------------------------------------------

type (
	Sorted_ []Op              // provides Len and Swap methods to sort operators
	ByName  struct{ Sorted_ } // lexicographically
	ByLen   struct{ Sorted_ } // descending length
	ByPrec  struct{ Sorted_ } // descending precedence
	ByTyp   struct{ Sorted_ } // ascending type
)

func (ops Sorted_) Len() int      { return len(ops) }
func (ops Sorted_) Swap(i, j int) { ops[i], ops[j] = ops[j], ops[i] }

func (ops ByName) Less(i, j int) bool {
	return ops.Sorted_[i].Name < ops.Sorted_[j].Name
}

func (ops ByLen) Less(i, j int) bool {
	return len(ops.Sorted_[i].Name) < len(ops.Sorted_[j].Name)
}

func (ops ByPrec) Less(i, j int) bool {
	return ops.Sorted_[i].Prec < ops.Sorted_[j].Prec
}

func (ops ByTyp) Less(i, j int) bool {
	return ops.Sorted_[i].Typ < ops.Sorted_[j].Typ
}

// Default Operators
// --------------------------------------------------

var defaultOps = OpTable{
	parent: nil,
	tab: map[string][]Op{
		"-->":   {{1200, XFX, "-->"}},
		"-":     {{500, YFX, "-"}, {200, FY, "-"}},
		"->":    {{1050, XFY, "->"}},
		",":     {{1000, XFY, ","}},
		";":     {{1100, XFY, ";"}},
		":-":    {{1200, XFX, ":-"}, {1200, FX, ":-"}},
		":":     {{600, XFY, ":"}},
		":<":    {{700, XFX, ":<"}},
		":=":    {{990, XFX, ":="}},
		"?":     {{500, FX, "?"}},
		".":     {{100, YFX, "."}},
		"@<":    {{700, XFX, "@<"}},
		"@=<":   {{700, XFX, "@=<"}},
		"@>":    {{700, XFX, "@>"}},
		"@>=":   {{700, XFX, "@>="}},
		"*->":   {{1050, XFY, "*->"}},
		"*":     {{400, YFX, "*"}},
		"**":    {{200, XFX, "**"}},
		"/":     {{400, YFX, "/"}},
		"//":    {{400, YFX, "//"}},
		"/\\":   {{500, YFX, "/\\"}},
		"\\":    {{200, FY, "\\"}},
		"\\/":   {{500, YFX, "\\/"}},
		"\\+":   {{900, FY, "\\+"}},
		"\\=":   {{700, XFX, "\\="}},
		"\\=@=": {{700, XFX, "\\=@="}},
		"\\==":  {{700, XFX, "\\=="}},
		"^":     {{200, XFY, "^"}},
		"+":     {{500, YFX, "+"}, {200, FY, "+"}},
		"<":     {{700, XFX, "<"}},
		"<<":    {{400, YFX, "<<"}},
		"=:=":   {{700, XFX, "=:="}},
		"=..":   {{700, XFX, "=.."}},
		"=":     {{700, XFX, "="}},
		"=@=":   {{700, XFX, "=@="}},
		"=\\=":  {{700, XFX, "=\\="}},
		"=<":    {{700, XFX, "=<"}},
		"==":    {{700, XFX, "=="}},
		">:<":   {{700, XFX, ">:<"}},
		">":     {{700, XFX, ">"}},
		">=":    {{700, XFX, ">="}},
		">>":    {{400, YFX, ">>"}},
		"|":     {{1100, XFY, "|"}},
		"$":     {{1, FX, "$"}},
		"as":    {{700, XFX, "as"}},
		"div":   {{400, YFX, "div"}},
		"is":    {{700, XFX, "is"}},
		"mod":   {{400, YFX, "mod"}},
		"rdiv":  {{400, YFX, "rdiv"}},
		"rem":   {{400, YFX, "rem"}},
		"xor":   {{500, YFX, "xor"}},
	},
}
