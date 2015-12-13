package parse

import "fmt"

// SyntaxError is the type of all errors produced by the parser.
type SyntaxError struct {
	Err string // description
	Tok Token  // cause
}

func (err SyntaxError) Error() string {
	return err.Err
}

func errorf(name string, tok Token, format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s:%d:%d: %s", name, tok.LineNo, tok.ColNo, msg)
}

func unexpected(name string, found Token, expected TokType) (err SyntaxError) {
	return SyntaxError{
		Err: errorf(name, found, "expected %v, found %v", expected, found),
		Tok: found,
	}
}

func priorityClash(name string, culprit Token) (err SyntaxError) {
	return SyntaxError{
		Err: errorf(name, culprit, "operator priority clash: %v", culprit),
		Tok: culprit,
	}
}

func ambiguous(name string, culprit Token) (err SyntaxError) {
	return SyntaxError{
		Err: errorf(name, culprit, "ambigous operator: %v", culprit),
		Tok: culprit,
	}
}

func compositeErr(errs ...error) (err SyntaxError) {
	for i := range errs {
		err.Err += errs[i].Error()
		if i != len(errs)-1 {
			err.Err += "\n"
		}
	}
	return err
}
