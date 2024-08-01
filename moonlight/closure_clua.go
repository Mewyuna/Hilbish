//go:build midnight
package moonlight

import (
	"fmt"
)

type Callable interface{
	Continuation(*Runtime, Cont) Cont
}

type Closure struct{
	refIdx int // so since we cant store the actual lua closure,
	// we need a index to the ref in the lua registry... or something like that.
}

func (mlr *Runtime) ClosureArg(c *GoCont, num int) (*Closure, error) {
	fmt.Println("type at ", num, "is", mlr.state.LTypename(num))
	
	return &Closure{
		refIdx: -1,
	}, nil
}

/*
func (c *Closure) Continuation(mlr *Runtime, c Cont) Cont {
}
*/
