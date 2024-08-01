//go:build !midnight
package moonlight

import (
	rt "github.com/arnodel/golua/runtime"
)

var NilValue = rt.NilValue

type Value = rt.Value
type ValueType = rt.ValueType
const (
	IntType = rt.IntType
	StringType = rt.StringType
	FunctionType = rt.FunctionType
	TableType = rt.TableType
)

func Type(v Value) ValueType {
	return ValueType(v.Type())
}

func StringValue(str string) Value {
	return rt.StringValue(str)
}

func IntValue(i int64) Value {
	return rt.IntValue(i)
}

func BoolValue(b bool) Value {
	return rt.BoolValue(b)
}

func TableValue(t *Table) Value {
	return rt.TableValue(t.lt)
}

func ToString(v Value) string {
	return v.AsString()
}

func ToTable(v Value) *Table {
	return convertToMoonlightTable(v.AsTable())
}

func AsValue(v interface{}) Value {
	return rt.AsValue(v)
}
