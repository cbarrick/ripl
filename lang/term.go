package lang

import (
	"fmt"
	"math/big"
)

type Term interface {
	// fmt.Stringer
}

// Compound
// --------------------------------------------------

type Compound struct {
	Funct string
	Args  []Term
}

func (c Compound) String() (str string) {
	str = c.Funct
	if len(c.Args) > 0 {
		str += "("
		for i := range c.Args {
			str += fmt.Sprint(c.Args[i])
			if i < len(c.Args)-1 {
				str += ","
			} else {
				str += ")"
			}
		}
	}
	return str
}

// List
// --------------------------------------------------

type List struct {
	Vals []Term
	Tail Term
}

func (l List) String() (str string) {
	str += "["
	for i := range l.Vals {
		str += fmt.Sprint(l.Vals[i])
		if i < len(l.Vals)-1 {
			str += ","
		}
	}
	for t := l.Tail; t != nil; {
		tlist, ok := t.(List)
		if ok {
			for i := range tlist.Vals {
				str += "," + fmt.Sprint(tlist.Vals[i])
			}
			t = tlist.Tail
		} else {
			str += "|" + fmt.Sprint(t)
			break
		}
	}
	str += "]"
	return str
}

// Num
// --------------------------------------------------

type Num struct {
	big.Rat
}

func (n Num) String() string {
	f, _ := n.Rat.Float64()
	return fmt.Sprint(f)
}

// Variable
// --------------------------------------------------

type Variable uint

func (v Variable) String() string {
	return fmt.Sprintf("_V%d", v)
}
