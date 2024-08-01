//go:build !midnight
package moonlight

import (
	rt "github.com/arnodel/golua/runtime"
)

type GoCont struct{
	cont *rt.GoCont
	thread *rt.Thread
}

type Cont = rt.Cont
type Closure = rt.Closure

func (gc *GoCont) Next() Cont {
	return gc.cont.Next()
}
