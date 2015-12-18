package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cbarrick/ripl/lang"
	"github.com/cbarrick/ripl/lang/op"
	"github.com/cbarrick/ripl/lang/parse"
)

func main() {
	var t lang.Term
	var err error
	parser := parse.File("stdin", os.Stdin, op.DefaultOps())
	for t, err = parser.NextClause(); err != io.EOF; {
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(t)
		}
		t, err = parser.NextClause()
	}
}
