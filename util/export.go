package util

import (
	"hilbish/moonlight"

	rt "github.com/arnodel/golua/runtime"
)

// LuaExport represents a Go function which can be exported to Lua.
type LuaExport struct {
	Function moonlight.GoFunctionFunc
	ArgNum int
	Variadic bool
}

// SetExports puts the Lua function exports in the table.
func SetExports(rtm *rt.Runtime, tbl *rt.Table, exports map[string]LuaExport) {
	/*
	for name, export := range exports {
		rtm.SetEnvGoFunc(tbl, name, export.Function, export.ArgNum, export.Variadic)
	}
	*/
}
