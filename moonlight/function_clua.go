//go:build midnight
package moonlight

import (
	"fmt"

	"github.com/aarzilli/golua/lua"
)

type GoFunctionFunc struct{
	cf lua.LuaGoFunction
}

func (gf GoFunctionFunc) Continuation(mlr *Runtime, c Cont) Cont {
	return &GoCont{
		f: gf,
		vals: []Value{},
	}
}

func (mlr *Runtime) CheckNArgs(c *GoCont, num int) error {
	args := mlr.state.GetTop()
	if args < num {
		return fmt.Errorf("%d arguments needed", num)
	}

	return nil
}

func (mlr *Runtime) Check1Arg(c *GoCont) error {
	return mlr.CheckNArgs(c, 1)
}

func (mlr *Runtime) StringArg(c *GoCont, num int) (string, error) {
	return mlr.state.CheckString(num + 1), nil
}

func (mlr *Runtime) Arg(c *GoCont, num int) Value {
	return c.vals[num]
}

func (mlr *Runtime) GoFunction(fun GoToLuaFunc) GoFunctionFunc {
	return GoFunctionFunc{
		cf: func(L *lua.State) int {
			cont, err := fun(mlr, &GoCont{})
			if err != nil {
				L.RaiseError(err.Error())
				return 0
			}

			for _, val := range cont.(*GoCont).vals {
				switch Type(val) {
					case StringType:
						L.PushString(val.AsString())
				}
			}

			return len(cont.(*GoCont).vals)
		},
	}
}
