package term

import "fmt"

type Compound struct {
	Funct string
	Args  []Term
}

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

func (c Compound) BreadthFirst(f func(int, Term)) {
	var i int
	ch := make(chan Term)
	go c.bfs(ch)
	for t := range ch {
		f(i, t)
		i++
	}
}

func (c Compound) bfs(ch chan Term) {
	ch <- c
	subs := make([]chan Term, 0, len(c.Args))
	for _, arg := range c.Args {
		if comp, ok := arg.(Compound); ok {
			sub := make(chan Term)
			subs = append(subs, sub)
			go comp.bfs(sub)
			ch <- <-sub
		} else {
			ch <- arg
		}
	}
	i := 0
	for len(subs) > 0 {
		val := <-subs[i]
		if val == nil {
			subs = append(subs[:i], subs[i+1:]...)
		} else {
			ch <- val
			i += 1
			i %= len(subs)
		}
	}
	close(ch)
}
