package types

import (
	"bytes"
	"fmt"
	"unicode"
)

type Variable string

func (*Variable) Type() ValueType {
	return VariableTyp
}

func (v *Variable) String() string {
	return string(*v)
}

func (v *Variable) Scan(state fmt.ScanState, verb rune) (err error) {
	var r rune
	var tok []byte
	buf := new(bytes.Buffer)
	if r, _, err = state.ReadRune(); err == nil {
		if r == '_' || unicode.IsUpper(r) {
			if _, err = buf.WriteRune(r); err == nil {
				if tok, err = state.Token(false, nil); err == nil {
					buf.Write(tok)
				}
			}
		}
	}
	*v = Variable(buf.String())
	return err
}
