package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cbarrick/ripl/lang"
	"github.com/cbarrick/ripl/lang/parse"
)

func main() {
	var t lang.Term
	var err error
	ops := lang.DefaultOps()
	lexer := parse.Lex(os.Stdin, ops)
	parser := parse.Parse("stdin", lexer, ops)
	for t, err = parser.Read(); err != io.EOF; {
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(t)
		}
		t, err = parser.Read()
	}
}
