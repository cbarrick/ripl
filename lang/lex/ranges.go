package lex

import "unicode"

// ASCIISymbols gives the valid ascii characters for a symbol functor.
const ASCIISymbols = "~`!@#$%^&*_-+=|\\:;<,>.?/"

// Symbols gives the valid runes for a symbol functor.
var Symbols = []*unicode.RangeTable{
	unicode.Symbol,
	unicode.Pc, // punctuation, connector (contains '_')
	unicode.Pd, // punctuation, dash
	unicode.Po, // punctuation, other (contains '!', and ',')
}

// ASCIILetters gives the valid ascii characters for a letter and number functor.
const ASCIILetters = "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz_0123456789"

// Letters gives the valid runes for a letter and number functor.
var Letters = []*unicode.RangeTable{
	unicode.Letter,
	unicode.Number,
	unicode.Pc, // punctuation, connector (contains '_')
}
