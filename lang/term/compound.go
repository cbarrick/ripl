package term

import "fmt"

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
