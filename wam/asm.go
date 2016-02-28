package wam

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// String returns WAM assembly for the program.
func (p Program) String() (str string) {
	var buf bytes.Buffer
	buf.WriteString(".data\n")
	for c, id := range p.consts {
		buf.WriteByte('\t')
		buf.WriteString(strconv.Itoa(int(id)))
		buf.WriteString(": ")
		switch c.(type) {
		case int:
			buf.WriteString("int ")
			buf.WriteString(strconv.Itoa(c.(int)))
		case float64:
			buf.WriteString("float ")
			buf.WriteString(strconv.FormatFloat(c.(float64), 'g', -1, 64))
		case string:
			buf.WriteString("funct ")
			buf.WriteString(c.(string))
		}
		buf.WriteByte('\n')
	}
	buf.WriteString(".text\n")
	for _, i := range p.code {
		buf.WriteByte('\t')
		buf.WriteString(i.String())
		buf.WriteByte('\n')
	}
	return buf.String()
}

// Scan builds the program from WAM assembly.
func (p *Program) Scan(state fmt.ScanState, verb rune) (err error) {
	for {
		var section string
		state.SkipSpace()
		_, err = fmt.Fscanf(state, ".%v", &section)
		if err == nil {
			switch section {
			case "text":
				err = p.scanText(state)
			case "data":
				err = p.scanData(state)
			default:
				return fmt.Errorf("unknown section %q", section)
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// scanText repetedly scans instructions
// and adds them to the program's code segment
func (p *Program) scanText(state fmt.ScanState) (err error) {
	for {
		var (
			i instruct
			r rune
		)
		state.SkipSpace()
		if r, _, err = state.ReadRune(); err != nil {
			return err
		}
		if err = state.UnreadRune(); err != nil {
			return err
		}
		if r == '.' {
			return err
		}
		if _, err = fmt.Fscanf(state, "%v", &i); err != nil {
			return err
		} else {
			p.code = append(p.code, i)
			continue
		}
	}
}

// scanData repetedly scans constants
// and adds them to the program's constants
func (p *Program) scanData(state fmt.ScanState) (err error) {
	for {
		var (
			id cid
			c  constant
			r  rune
		)
		state.SkipSpace()
		if r, _, err = state.ReadRune(); err != nil {
			return err
		}
		if err = state.UnreadRune(); err != nil {
			return err
		}
		if r == '.' {
			return err
		}
		if id, c, err = scanConst(state); err != nil {
			return err
		} else {
			p.consts[c] = id
			continue
		}
	}
}

func scanConst(state fmt.ScanState) (id cid, c constant, err error) {
	var typ string

	// scan the id and type
	if _, err = fmt.Fscanf(state, "%v: %v ", &id, &typ); err != nil {
		return id, c, err
	}

	// the format of the data depends on the type
	switch typ {
	case "int":
		var val int
		_, err = fmt.Fscanf(state, "%v", val)
		return id, val, err

	case "float":
		var val float64
		_, err = fmt.Fscanf(state, "%v", val)
		return id, val, err

	case "funct":
		state.SkipSpace()
		var r rune
		if r, _, err = state.ReadRune(); err != nil {
			return id, c, err
		}

		// quoted
		if r == '\'' {
			var buf bytes.Buffer
			r, _, err = state.ReadRune()
			for r != '\'' && err == nil {
				buf.WriteRune(r)
				r, _, err = state.ReadRune()
			}
			if err != nil {
				return id, c, err
			}
			return id, buf.String(), err
		}

		// unquoted
		var val string
		if err = state.UnreadRune(); err != nil {
			return id, val, err
		}
		if _, err = fmt.Fscanf(state, "%v", &val); err != nil {
			return id, val, err
		}
		return id, val, err

	default:
		return id, c, fmt.Errorf("unknown type %q", typ)
	}
}

// Instructions:
// --------------------------------------------------

// String returns WAM assembly for the instruction.
func (i instruct) String() string {
	switch i.opcode {
	case put_struct:
		return fmt.Sprintf("put_struct %v/%v, $%v", i.cid, i.arity, i.reg1)

	case get_struct:
		return fmt.Sprintf("get_struct %v/%v, $%v", i.cid, i.arity, i.reg1)

	case set_var:
		return fmt.Sprintf("set_var $%v", i.reg1)

	case set_val:
		return fmt.Sprintf("set_val $%v", i.reg1)

	case unify_var:
		return fmt.Sprintf("unify_var $%v", i.reg1)

	case unify_val:
		return fmt.Sprintf("unify_val $%v", i.reg1)

	default:
		return fmt.Sprintf("%#v", i)
	}
}

// Scan reads an instruction in WAM assembly.
func (i *instruct) Scan(state fmt.ScanState, verb rune) (err error) {
	var opcode string
	_, err = fmt.Fscanf(state, "%s", &opcode)
	if err != nil {
		return err
	}
	state.SkipSpace()
	switch opcode {
	case "put_struct":
		i.opcode = put_struct
		_, err = fmt.Fscanf(state, "%d/%d, $%d", &i.cid, &i.arity, &i.reg1)
		return err

	case "get_struct":
		i.opcode = get_struct
		_, err = fmt.Fscanf(state, "%d/%d, $%d", &i.cid, &i.arity, &i.reg1)
		return err

	case "set_var":
		i.opcode = set_var
		_, err = fmt.Fscanf(state, "$%d", &i.reg1)
		return err

	case "set_val":
		i.opcode = set_val
		_, err = fmt.Fscanf(state, "$%d", &i.reg1)
		return err

	case "unify_var":
		i.opcode = unify_var
		_, err = fmt.Fscanf(state, "$%d", &i.reg1)
		return err

	case "unify_val":
		i.opcode = unify_val
		_, err = fmt.Fscanf(state, "$%d", &i.reg1)
		return err

	default:
		return fmt.Errorf("unknown opcode %q", opcode)
	}
}
