//go:build !midnight
package moonlight

import (
	rt "github.com/arnodel/golua/runtime"
)

type Table struct{
	lt *rt.Table
}

func NewTable() *Table {
	return &Table{
		lt: rt.NewTable(),
	}
}

func (t *Table) Get(val Value) Value {
	return t.lt.Get(val)
}

func (t *Table) SetField(key string, value Value) {
	t.lt.Set(rt.StringValue(key), value)
}

func (t *Table) Set(key Value, value Value) {
	t.lt.Set(key, value)
}

func ForEach(tbl *Table, cb func(key Value, val Value)) {
	nextVal := rt.NilValue
	for {
		key, val, _ := tbl.lt.Next(nextVal)
		if key == rt.NilValue {
			break
		}
		nextVal = key

		cb(Value(key), Value(val))
	}
}

func (mlr *Runtime) GlobalTable() *Table {
	return &Table{
		lt: mlr.rt.GlobalEnv(),
	}
}

func convertToMoonlightTable(t *rt.Table) *Table {
	return &Table{
		lt: t,
	}
}

func TryTable(v Value) (*Table, bool) {
	t, ok := v.TryTable()

	return convertToMoonlightTable(t), ok
}
