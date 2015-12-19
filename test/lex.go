package main

import (
	"fmt"
	"os"

	"github.com/cbarrick/ripl/lang/parse"
)

func main() {
	var (
		lexer = parse.Lex("stdin", os.Stdin, parse.DefaultOps())
		tok   parse.Token
		err   error
	)
	for tok, err = lexer.NextToken(); err == nil; tok, err = lexer.NextToken() {
		fmt.Println(tok)
	}
	fmt.Println(err)
	lexer.Close()
}
