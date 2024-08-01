package moonlight

type GoToLuaFunc func(mlr *Runtime, c *GoCont) (Cont, error)
