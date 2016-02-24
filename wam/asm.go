package wam

import "fmt"

// String returns WAM assembly for the program.
func (p *Program) String() (str string) {
	str += "CONSTANTS"
	for _, t := range p.constants {
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

// String returns the instruction as WAM assembly.
func (i instruct) String() string {
	switch i.opcode {
	case put_struct:
		return fmt.Sprintf("put_struct %v/%d, $%d", i.cid, i.arity, i.reg1)
	case get_struct:
		return fmt.Sprintf("get_struct %v/%d, $%d", i.cid, i.arity, i.reg1)
	case set_var:
		return fmt.Sprintf("set_var $%d", i.reg1)
	case set_val:
		return fmt.Sprintf("set_val $%d", i.reg1)
	case unify_var:
		return fmt.Sprintf("unify_var $%d", i.reg1)
	case unify_val:
		return fmt.Sprintf("unify_val $%d", i.reg1)
	default:
		return fmt.Sprintf("%#v", i)
	}
}
