package symbol

// Interface is the common interface for all lexical symbols.
type Interface interface {
	Type() Type
	Hash() int64
	Cmp(s Interface) int
	String() string
}
