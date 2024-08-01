//go:build !midnight
package moonlight

import (
	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/lib/packagelib"
)

type Loader func(*Runtime) Value

func (mlr *Runtime) LoadLibrary(ldr Loader, name string) {
	goluaLoader := packagelib.Loader{
		Load: func(rt *rt.Runtime) (rt.Value, func()) {
			val := ldr(specificRuntimeToGeneric(rt))

			return val, nil
		},
		Name: name,
	}

	lib.LoadLibs(mlr.rt, goluaLoader)
}
