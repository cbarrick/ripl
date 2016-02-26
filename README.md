Ripl
==================================================

Ripl will be a Prolog implementation in Go.

Dependencies
--------------------------------------------------
- http://golang.org/x/text/unicode/norm is used for unicode normalization.

Checklist
--------------------------------------------------
- [X] Prolog Parser
	- [X] UTF-8 support
	- [X] Arbitrary operators
- [ ] Prolog -> WAM compiler
	- [x] Read the WAM book: http://wambook.sourceforge.net/
	- [x] L0 (unification)
	- [ ] L1 (procedure calls)
	- [ ] L2 (flat resolution)
	- [ ] L3 (Prolog)
	- [ ] Cuts and optimizations
- [ ] WAM evaluator
- [ ] Database/modules
- [ ] REPL/frontend
- [ ] etc.
