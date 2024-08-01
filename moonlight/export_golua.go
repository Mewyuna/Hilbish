//go:build !midnight
package moonlight

func (mlr *Runtime) SetExports(tbl *Table, exports map[string]Export) {
	for name, export := range exports {
		mlr.rt.SetEnvGoFunc(tbl.lt, name, mlr.GoFunction(export.Function), export.ArgNum, export.Variadic)
	}
}
