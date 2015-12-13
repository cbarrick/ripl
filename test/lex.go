package main

import (
	"fmt"
	"os"

	"github.com/cbarrick/ripl/lang"
	"github.com/cbarrick/ripl/lang/parse"
)

func main() {
	var (
		lexer = parse.Lex(os.Stdin, lang.DefaultOps())
		tok   parse.Token
		err   error
	)
	for tok, err = lexer.Read(); err == nil; tok, err = lexer.Read() {
		fmt.Println(tok)
	}
	fmt.Println(err)
}
