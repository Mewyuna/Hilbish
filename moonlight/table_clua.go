//go:build midnight
package moonlight

//import "github.com/aarzilli/golua/lua"

type Table struct{
	refIdx int
}

func NewTable() *Table {
	return &Table{
		refIdx: -1,
	}
}

func (t *Table) Get(val Value) Value {
	return NilValue
}

func (t *Table) SetField(key string, value Value) {
}

func (t *Table) Set(key Value, value Value) {
}

func ForEach(tbl *Table, cb func(key Value, val Value)) {
}

func (mlr *Runtime) GlobalTable() *Table {
	return &Table{
		refIdx: -1,
	}
}

func ToTable(v Value) *Table {
	return &Table{
		refIdx: -1,
	}
}

func TryTable(v Value) (*Table, bool) {
	return nil, false
}

func (t *Table) setRefIdx(mlr *Runtime, i idx) {
	t.refIdx = mlr.state.Ref(i)
}
