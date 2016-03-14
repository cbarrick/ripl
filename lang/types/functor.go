package types

import (
	"bytes"
	"fmt"
	"strconv"
)

type Functor bytes.Buffer

func (*Functor) Type() ValueType {
	return FunctorTyp
}

func (f *Functor) String() string {
	return (*bytes.Buffer)(f).String()
}

func (f *Functor) Scan(state fmt.ScanState, verb rune) (err error) {
	var b = (*bytes.Buffer)(f)
	var tok []byte
	var r rune
	r, _, err = state.ReadRune()
	if err != nil {
		return err
	}
	if r == '\'' {
		return f.scanQuote(state)
	}
	b.WriteRune(r)
	if tok, err = state.Token(false, nil); err == nil {
		b.Write(tok)
	}
	return err
}

func (f *Functor) scanQuote(state fmt.ScanState) (err error) {
	var b = (*bytes.Buffer)(f)
	var r rune
	r, _, err = state.ReadRune()
	for r != '\'' && err == nil {
		if r == '\\' {
			if r, _, err = state.ReadRune(); err == nil {
				r, _, _, err = strconv.UnquoteChar("\\"+string(r), 0)
			}
		}
		b.WriteRune(r)
		r, _, err = state.ReadRune()
	}
	return err
}
