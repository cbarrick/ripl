package term

import "fmt"

type Compound struct {
	Funct string
	Args  []Term
}

// String returns the cannonical representation of the compound.
func (c Compound) String() (str string) {
	str = c.Funct
	if len(c.Args) > 0 {
		str += "("
		for i := range c.Args {
			str += fmt.Sprint(c.Args[i])
			if i < len(c.Args)-1 {
				str += ","
			} else {
				str += ")"
			}
		}
	}
	return str
}

// TopDown returns a channel that yields the subterms of c in top-down,
// left-to-right level-order.
func (c Compound) TopDown() chan Term {
	ch := make(chan Term)
	go c.topDown(ch)
	return ch
}

func (c Compound) topDown(ch chan Term) {
	defer close(ch)

	const (
		init = 16 // initial size of the queue's buffer
	)

	var (
		queue = make(chan Term, init)
	)

	// ensures that there is room for n more terms in the queue
	ensure := func(n int) {
		l := len(queue)
		c := cap(queue)
		if c < l+n {
			for c < n {
				c = c << 1
			}
			q := make(chan Term, c)
			for 0 < l {
				q <- <-queue
				l--
			}
			close(queue)
			queue = q
		}
	}

	queue <- c
	for len(queue) > 0 {
		t := <-queue
		ch <- t
		if t, ok := t.(Compound); ok {
			ensure(len(t.Args))
			for i := range t.Args {
				queue <- t.Args[i]
			}
		}
	}
}

// BottomUp returns a channel that yields the subterms of c in bottom-up,
// left-to-right level-order.
func (c Compound) BottomUp() chan Term {
	ch := make(chan Term)
	go c.bottomUp(ch)
	return ch
}

func (c Compound) bottomUp(ch chan Term) {
	defer close(ch)

	var (
		terms = [][]Term{[]Term{c}}
		depth int
	)

	for {
		var level []Term
		for i := range terms[depth] {
			if t, ok := terms[depth][i].(Compound); ok {
				level = append(level, t.Args...)
			}
		}
		if level == nil {
			break
		}
		terms = append(terms, level)
		depth++
	}

	for 0 <= depth {
		for i := range terms[depth] {
			ch <- terms[depth][i]
		}
		depth--
	}
}
