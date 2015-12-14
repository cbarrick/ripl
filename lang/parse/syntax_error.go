package parse

import "fmt"

// SyntaxError is the type of all errors produced by the parser.
type SyntaxError struct {
	Err  string       // description
	Tok  Token        // cause
	Prev *SyntaxError // multiple errors form a linked list
}

func (err *SyntaxError) Error() string {
	if err.Prev == nil {
		return err.Err
	}
	prev := err.Prev.Error()
	return prev + "\n" + err.Err
}

func errorf(name string, tok Token, format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s:%d:%d: %s", name, tok.LineNo, tok.ColNo, msg)
}

func unexpected(name string, found Token, expected TokType) *SyntaxError {
	return &SyntaxError{
		Err: errorf(name, found, "expected %v, found %v", expected, found),
		Tok: found,
	}
}

func priorityClash(name string, culprit Token) *SyntaxError {
	return &SyntaxError{
		Err: errorf(name, culprit, "operator priority clash: %v", culprit),
		Tok: culprit,
	}
}

func ambiguousOp(name string, culprit Token) *SyntaxError {
	return &SyntaxError{
		Err: errorf(name, culprit, "ambigous operator: %v", culprit),
		Tok: culprit,
	}
}

func wrapErr(name string, tok Token, err error) *SyntaxError {
	return &SyntaxError{
		Err: errorf(name, tok, err.Error()),
		Tok: tok,
	}
}
