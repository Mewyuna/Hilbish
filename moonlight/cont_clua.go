//go:build midnight
package moonlight

type GoCont struct{
	vals []Value
	f GoFunctionFunc
}
type Cont interface{}

func (gc *GoCont) Next() Cont {
	return gc
}
