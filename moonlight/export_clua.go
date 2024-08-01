//go:build midnight
package moonlight

func (mlr *Runtime) SetExports(tbl *Table, exports map[string]Export) {
	for name, export := range exports {
		tbl.SetField(name, FunctionValue(mlr.GoFunction(export.Function)))
	}
}
