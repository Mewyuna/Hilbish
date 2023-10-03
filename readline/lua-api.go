package readline

import (
	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
)

func (rl *Instance) SetupLua(rtm *rt.Runtime) *rt.Table {
	exports := map[string]util.LuaExport{
		"insert": {rl.editorInsert, 1, false},
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, exports)

	return mod
}

// #interface editor
// insert(text)
// Inserts text into the line.
func (rl *Instance) editorInsert(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	text, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	rl.Insert(text)

	return c.Next(), nil
}

