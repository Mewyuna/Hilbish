//go:build midnight
package moonlight

type Value struct{
	iface interface{}
	relIdx int
	refIdx int
}

var NilValue = Value{nil, -1, -1}

type ValueType uint8
const (
	NilType ValueType = iota
	BoolType
	IntType
	StringType
	TableType
	FunctionType
	UnknownType
)

func Type(v Value) ValueType {
	return v.Type()
}

func BoolValue(b bool) Value {
	return Value{iface: b}
}

func IntValue(i int64) Value {
	return Value{iface: i}
}

func StringValue(str string) Value {
	return Value{iface: str}
} 

func TableValue(t *Table) Value {
	return Value{iface: t}
}

func FunctionValue(f Callable) Value {
	return NilValue
}

func AsValue(i interface{}) Value {
	if i == nil {
		return NilValue
	}

	switch v := i.(type) {
		case bool: return BoolValue(v)
		case int64: return IntValue(v)
		case string: return StringValue(v)
		case *Table: return TableValue(v)
		case Value: return v
		default:
			return Value{iface: i}
	}
}

func (v Value) Type() ValueType {
	if v.iface == nil {
		return NilType
	}

	switch v.iface.(type) {
		case bool: return BoolType
		case int: return IntType
		case string: return StringType
		case *Table: return TableType
		case *Closure: return FunctionType
		default: return UnknownType
	}
}

func (v Value) AsInt() int64 {
	return v.iface.(int64)
}

func (v Value) AsString() string {
	if v.Type() != StringType {
		panic("value type was not string")
	}

	return v.iface.(string)
}

func (v Value) AsTable() *Table {
	panic("Value.AsTable unimplemented in midnight")
}

func ToString(v Value) string {
	return v.AsString()
}

func (v Value) TypeName() string {
	switch v.iface.(type) {
		case bool: return "bool"
		case int: return "number"
		case string: return "string"
		case *Table: return "table"
		default: return "<unknown type>"
	}
}

func (v Value) TryBool() (n bool, ok bool) {
	n, ok = v.iface.(bool)
	return
}

func (v Value) TryInt() (n int, ok bool) {
	n, ok = v.iface.(int)
	return
}

func (v Value) TryString() (n string, ok bool) {
	n, ok = v.iface.(string)
	return
}
