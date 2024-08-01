//go:build midnight
package moonlight

func (mlr *Runtime) DoString(code string) (Value, error) {
	err := mlr.state.DoString(code)

	return NilValue, err
}

func (mlr *Runtime) DoFile(filename string) error {
	//return mlr.state.DoFile(filename)
	return nil
}
