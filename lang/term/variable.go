package term

import "fmt"

type Variable uint

func (v Variable) String() string {
	return fmt.Sprintf("_V%d", v)
}
