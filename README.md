Ripl
==================================================

Ripl is a parser for Prolog. One day it may become something more.

The programs `test/parse.go` and `test/lex.go` take in Prolog clauses on stdin and print the parse tree or lexemes to stdout.

Dependencies
--------------------------------------------------
- golang.org/x/text/unicode/norm is used for unicode normalization.

TODO:
--------------------------------------------------
- [ ] Force synchronized access to the operator table with a mutex.
- [ ] Make a todo list.
