package term

import "fmt"

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
