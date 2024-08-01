package moonlight

type Export struct{
	Function GoToLuaFunc
	ArgNum int
	Variadic bool
}
