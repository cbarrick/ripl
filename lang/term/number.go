package term

import (
	"fmt"
	"math/big"
)

type Num struct {
	big.Rat
}

func (n Num) String() string {
	f, _ := n.Rat.Float64()
	return fmt.Sprint(f)
}
