package wam

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// String returns WAM assembly for the program.
func (p Program) String() (str string) {
	str += "CONSTANTS"
	for _, t := range p.constants {
		str += fmt.Sprintf("\n\t%#v", t)
	}
	str += "\nTEXT"
	for _, ins := range p.code {
		str += fmt.Sprintf("\n\t%v", ins)
	}
	return str
}

// Scan reads a program in WAM assembly.
func (p *Program) Scan(state fmt.ScanState, verb rune) (err error) {
	readLine := func() (line []byte, err error) {
		line, err = state.Token(true, func(r rune) bool {
			return r != '\n'
		})
		return line, err
	}

	parseConst := func(buf []byte) (c constant, err error) {
		str := strings.Trim(string(buf), " \t\r\n")
		c, err = strconv.ParseInt(str, 0, 64)
		if err != nil {
			c, err = strconv.ParseFloat(str, 64)
			if err != nil {
				c, err = strconv.Unquote(str)
			}
		}
		return c, err
	}

	var buf []byte
	buf, err = readLine()
	switch {
	case strings.HasPrefix(string(buf), "CONSTANTS"):
		for err == nil {
			buf, err = readLine()
			if err == nil {
				var c constant
				c, err = parseConst(buf)
				if err == nil {
					p.cids[c] = cid(len(p.constants))
					p.constants = append(p.constants, c)
				} else {
					err = nil
					break
				}
			}
		}
		if !strings.HasPrefix(string(buf), "TEXT") {
			return fmt.Errorf("expecting text segment")
		}
		fallthrough

	case strings.HasPrefix(string(buf), "TEXT"):
		for {
			var i instruct
			state.SkipSpace()
			err = i.Scan(state, verb)
			if err != nil || i.opcode == eot {
				break
			}
			p.code = append(p.code, i)
		}
	}

	return err
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

// Scan reads an instruction in WAM assembly.
func (i *instruct) Scan(state fmt.ScanState, verb rune) error {
	// scans the next non-space rune
	// returns an error if the scanned rune is not r
	var expect func(rune) error
	expect = func(r rune) error {
		s, _, err := state.ReadRune()
		if err == nil && unicode.IsSpace(s) {
			return expect(r)
		} else if err == nil && r != s {
			err = fmt.Errorf("expected %q, found %q", r, s)
		}
		return err
	}

	// scans a 'cid/arity' pair
	functor := func() (funct cid, ar arity, err error) {
		var buf []byte
		var n uint64
		buf, err = state.Token(true, unicode.IsNumber)
		if err == nil {
			n, err = strconv.ParseUint(string(buf), 0, 16)
			funct = cid(n)
			if err == nil {
				err = expect('/')
				if err == nil {
					buf, err = state.Token(true, unicode.IsNumber)
					if err == nil {
						n, err = strconv.ParseUint(string(buf), 0, 8)
						ar = arity(n)
					}
				}
			}
		}
		return funct, ar, err
	}

	// scans a register argument '$n'
	register := func() (reg register, err error) {
		var buf []byte
		var n uint64
		err = expect('$')
		if err == nil {
			buf, err = state.Token(true, unicode.IsNumber)
			if err == nil {
				n, err = strconv.ParseUint(string(buf), 0, 8)
				reg = register(n)
			}
		}
		return reg, err
	}

	opcode, err := state.Token(true, nil)
	switch string(opcode) {
	case "put_struct":
		i.opcode = put_struct
		i.cid, i.arity, err = functor()
		if err == nil {
			err = expect(',')
			if err == nil {
				i.reg1, err = register()
			}
		}

	case "get_struct":
		i.opcode = get_struct
		i.cid, i.arity, err = functor()
		if err == nil {
			err = expect(',')
			if err == nil {
				i.reg1, err = register()
			}
		}

	case "set_var":
		i.opcode = set_var
		i.reg1, err = register()

	case "set_val":
		i.opcode = set_val
		i.reg1, err = register()

	case "unify_var":
		i.opcode = unify_var
		i.reg1, err = register()

	case "unify_val":
		i.opcode = unify_val
		i.reg1, err = register()

	default:
		if len(opcode) > 0 {
			err = fmt.Errorf("unknown opcode %v", opcode)
		}
	}

	return err
}
