package types

import (
	"fmt"
	"math/big"
)

type Number big.Rat

func (*Number) Type() ValueType {
	return NumberTyp
}

func (n *Number) String() string {
	x := (*big.Rat)(n)
	if x.IsInt() {
		return x.Num().String()
	}
	f, _ := x.Float64()
	return fmt.Sprint(f)
}

func (n *Number) Scan(state fmt.ScanState, verb rune) error {
	x := (*big.Rat)(n)
	_, err := fmt.Fscan(state, x)
	return err
}
