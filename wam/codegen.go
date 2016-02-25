package wam

import "github.com/cbarrick/ripl/lang/term"

// A Program is a bytecode program for Ripl's implementation of the Warren
// Abstract Machine (WAM).
type Program struct {
	code      []instruct
	constants []constant
	cids      map[constant]cid
}

// An instruct is a bytecode WAM instruction.
// It is carefully crafted to fit in a 64bit word.
type instruct struct {
	opcode
	arity
	cid
	reg1 register
	reg2 register
}

// Various low-level types.
type (
	opcode   uint8
	arity    uint8
	register uint16
	cid      uint16
	constant interface{}
)

// The list of known opcodes.
const (
	eot opcode = iota
	put_struct
	set_var
	set_val
	get_struct
	unify_var
	unify_val
)

func NewProg() *Program {
	return &Program{
		cids: make(map[constant]cid),
	}
}

// CompileQuery compiles a term as a query
// and appends the instructions to the program's code segment.
func (p *Program) CompileQuery(q term.Compound) {
	panic("not implemented")
}

// CompileHead compiles a term as a clause head
// and appends the instructions to the program's code segment.
func (p *Program) CompileHead(head term.Compound) {
	var (
		i    register
		reg  = i + 1
		vars = make(map[term.Variable]register)
	)

	for t := range head.TopDown() {
		switch t := t.(type) {
		case term.Compound:
			p.code = append(p.code, instruct{
				opcode: get_struct,
				arity:  arity(len(t.Args)),
				cid:    p.cidOf(t.Funct),
				reg1:   i,
			})
			for _, arg := range t.Args {
				if arg, ok := arg.(term.Variable); ok {
					if vars[arg] != 0 {
						p.code = append(p.code, instruct{
							opcode: unify_val,
							reg1:   vars[arg],
						})
						continue
					}
					vars[arg] = reg
				}
				p.code = append(p.code, instruct{
					opcode: unify_var,
					reg1:   reg,
				})
				reg++
			}
			i++

		case term.Variable:
			if vars[t] == i {
				i++
			}

		default:
			panic("unknow term type")
		}
	}
}

// cidOf returns the id of program constant c,
// adding c to the program if needed.
func (p *Program) cidOf(c constant) (id cid) {
	id = p.cids[c]
	if id == 0 {
		id = cid(len(p.cids))
		p.cids[c] = id
		p.constants = append(p.constants, c)
	}
	return id
}
