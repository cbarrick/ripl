package lex

// ConstError is the type of errors returned by this package.
type ConstError string

func (e ConstError) Error() string {
	return string(e)
}

// ErrBadEncoding occurs when the Prolog source is not valid utf8.
const ErrBadEncoding = ConstError("invalid encoding")

// ErrUnclosedQuote occurs when a quoted atom is not closed.
const ErrUnclosedQuote = ConstError("unclosed quote")
