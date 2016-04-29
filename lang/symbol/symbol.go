package symbol

// Symbol is the common interface for all lexical symbols.
type Symbol interface {
	Type() Type
	Hash() int64
	Cmp(s Symbol) int
	String() string
}
