package wam

import (
	"reflect"

	"github.com/cbarrick/ripl/lang/term"
)

type Program struct {
	code      []instruct
	constIDs  map[interface{}]uint16
	constVals []interface{}
}

type instruct struct {
	opcode uint8
	arity  uint8
	arg1   uint16
	arg2   uint16
}

const (
	put_struct uint8 = 1 + iota
	set_var
	set_val
	get_struct
	unify_var
	unify_val
)

func NewProg() Program {
	return Program{
		constIDs: make(map[interface{}]uint16),
	}
}

// CompileFact compiles a term as a fact
// and appends the instructions to the program's code segment.
func (p *Program) CompileFact(fact term.Compound) {
	var (
		i    uint16
		reg  = i + 1
		vars = make(map[term.Variable]uint16)
	)

	fact.BreadthFirst(func(_ int, t term.Term) {
		switch t := t.(type) {
		case term.Compound:
			p.code = append(p.code, instruct{
				opcode: get_struct,
				arity:  uint8(len(t.Args)),
				arg1:   p.Constant(t.Funct),
				arg2:   i,
			})
			for _, arg := range t.Args {
				if arg, ok := arg.(term.Variable); ok {
					if vars[arg] != 0 {
						p.code = append(p.code, instruct{
							opcode: unify_val,
							arg1:   vars[arg],
						})
						continue
					}
					vars[arg] = reg
				}
				p.code = append(p.code, instruct{
					opcode: unify_var,
					arg1:   reg,
				})
				reg++
			}
			i++

		case term.Variable:
			if vars[t] == i {
				i++
			}

		default:
			panic("cannot compile type " + reflect.TypeOf(t).Name())
		}
	})
}

// Constant returns the id of program constant c,
// adding c to the program if needed.
func (p *Program) Constant(c interface{}) (id uint16) {
	id = p.constIDs[c]
	if id == 0 {
		id = uint16(len(p.constIDs))
		p.constIDs[c] = id
		p.constVals = append(p.constVals, c)
	}
	return id
}

// GetConstant returns the program constant identified by id.
func (p *Program) GetConstant(id uint64) (c interface{}) {
	return p.constVals[id-1]
}
