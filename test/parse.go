package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cbarrick/ripl/lang/term"
	"github.com/cbarrick/ripl/lang/parse"
)

func main() {
	var t term.Term
	var err error
	ops := parse.DefaultOps()
	parser := parse.File("stdin", os.Stdin, ops)
	for t, err = parser.NextClause(); err != io.EOF; {
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(t)
		}
		t, err = parser.NextClause()
	}
}
