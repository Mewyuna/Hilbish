//go:build !midnight
package moonlight

import (
	"os"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/lib/debuglib"
)

type Runtime struct{
	rt *rt.Runtime
}

func NewRuntime() *Runtime {
	r := rt.New(os.Stdout)
	r.PushContext(rt.RuntimeContextDef{
		MessageHandler: debuglib.Traceback,
	})
	lib.LoadAll(r)

	return specificRuntimeToGeneric(r)
}

func specificRuntimeToGeneric(rtm *rt.Runtime) *Runtime {
	rr := Runtime{
		rt: rtm,
	}

	return &rr
}

func (mlr *Runtime) UnderlyingRuntime() *rt.Runtime {
	return mlr.rt
}

// Push will push a Lua value onto the stack.
func (mlr *Runtime) Push(c *GoCont, v Value) {
	c.cont.Push(c.thread.Runtime, v)
}

func (mlr *Runtime) PushNext1(c *GoCont, v Value) Cont {
	return c.cont.PushingNext1(c.thread.Runtime, v)
}

func (mlr *Runtime) Call1(val Value, args ...Value) (Value, error) {
	return rt.Call1(mlr.rt.MainThread(), val, args...)
}
