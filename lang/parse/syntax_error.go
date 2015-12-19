package parse

import "fmt"

// SyntaxError is the type of all errors produced by the parser. Underlying
// IO errors are wrapped in syntax errors before being passed up, except io.EOF.
type SyntaxError struct {
	Err  error        // underlying error
	Tok  Token        // cause of the error
	Prev *SyntaxError // multiple errors form a stack
}

// Error returns an error message containing listing errors in the stack.
func (e *SyntaxError) Error() (msg string) {
	msg = fmt.Sprintf("%s:%d:%d: %s", e.Tok.Name, e.Tok.LineNo, e.Tok.ColNo, e.Err.Error())
	if e.Prev == nil {
		return msg
	}
	prev := e.Prev.Error()
	return prev + "\n" + msg
}

func unexpected(found Token, expected TokType) *SyntaxError {
	return &SyntaxError{
		Err: fmt.Errorf("expected %v, found %v", expected, found),
		Tok: found,
	}
}

func priorityClash(culprit Token) *SyntaxError {
	return &SyntaxError{
		Err: fmt.Errorf("operator priority clash: %v", culprit),
		Tok: culprit,
	}
}

func ambiguousOp(culprit Token) *SyntaxError {
	return &SyntaxError{
		Err: fmt.Errorf("ambigous operator: %v", culprit),
		Tok: culprit,
	}
}

func wrapErr(tok Token, err error) *SyntaxError {
	return &SyntaxError{
		Err: err,
		Tok: tok,
	}
}
