package wam

import "fmt"

func (i instruct) String() string {
	switch i.opcode {
	case put_struct:
		return fmt.Sprintf("put_struct %v/%d, $%d", i.arg1, i.arity, i.arg2)
	case get_struct:
		return fmt.Sprintf("get_struct %v/%d, $%d", i.arg1, i.arity, i.arg2)
	case set_var:
		return fmt.Sprintf("set_var $%d", i.arg1)
	case set_val:
		return fmt.Sprintf("set_val $%d", i.arg1)
	case unify_var:
		return fmt.Sprintf("unify_var $%d", i.arg1)
	case unify_val:
		return fmt.Sprintf("unify_val $%d", i.arg1)
	default:
		return fmt.Sprintf("%#v", i)
	}
}

func (p *Program) String() (str string) {
	str += "CONSTANTS"
	for _, t := range p.constVals {
		str += "\n\t"
		if t, ok := t.(string); ok {
			str += "\"" + t + "\""
		} else {
			str += t
		}
	}
	str += "\nTEXT"
	for _, ins := range p.code {
		str += "\n\t" + ins.String()
	}
	return str
}
