//go:build midnight
package moonlight

import (
	"github.com/aarzilli/golua/lua"
)

type Loader func(*Runtime) Value

func (mlr *Runtime) LoadLibrary(ldr Loader, name string) {
	cluaLoader := func (L *lua.State) int {
		mlr.pushToState(ldr(mlr))

		return 1
	}

	mlr.state.GetGlobal("package")
	mlr.state.GetField(-1, "loaded")
	mlr.state.PushGoFunction(cluaLoader)
	mlr.state.SetField(-2, name)
}
