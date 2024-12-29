package snail

import (
	"fmt"
	"strings"

	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/lib/packagelib"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var snailMetaKey = rt.StringValue("hshsnail")
var Loader = packagelib.Loader{
	Load: loaderFunc,
	Name: "snail",
}

func loaderFunc(rtm *rt.Runtime) (rt.Value, func()) {
	snailMeta := rt.NewTable()
	snailMethods := rt.NewTable()
	snailFuncs := map[string]util.LuaExport{
		"run": {srun, 1, false},
	}
	util.SetExports(rtm, snailMethods, snailFuncs)

	snailIndex := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		arg := c.Arg(1)
		val := snailMethods.Get(arg)

		return c.PushingNext1(t.Runtime, val), nil
	}
	snailMeta.Set(rt.StringValue("__index"), rt.FunctionValue(rt.NewGoFunction(snailIndex, "__index", 2, false)))
	rtm.SetRegistry(snailMetaKey, rt.TableValue(snailMeta))

	exports := map[string]util.LuaExport{
		"new": util.LuaExport{snew, 0, false},
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, exports)

	return rt.TableValue(mod), nil
}

func snew(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	s := New(t.Runtime)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(snailUserData(s))), nil
}

func srun(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}

	s, err := snailArg(c, 0)
	if err != nil {
		return nil, err
	}

	cmd, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}

	var newline bool
	var cont bool
	var luaErr rt.Value = rt.NilValue
	exitCode := 0
	bg, _, _, err := s.Run(cmd, nil)
	if err != nil {
		if syntax.IsIncomplete(err) {
			/*
			if !interactive {
				return cmdString, 126, false, false, err
			}
			*/
			if strings.Contains(err.Error(), "unclosed here-document") {
				newline = true
			}
			cont = true
		} else {
			if code, ok := interp.IsExitStatus(err); ok {
				exitCode = int(code)
			} else {
				luaErr = rt.StringValue(err.Error())
			}
		}
	}
	runnerRet := rt.NewTable()
	runnerRet.Set(rt.StringValue("input"), rt.StringValue(cmd))
	runnerRet.Set(rt.StringValue("exitCode"), rt.IntValue(int64(exitCode)))
	runnerRet.Set(rt.StringValue("continue"), rt.BoolValue(cont))
	runnerRet.Set(rt.StringValue("newline"), rt.BoolValue(newline))
	runnerRet.Set(rt.StringValue("err"), luaErr)

	runnerRet.Set(rt.StringValue("bg"), rt.BoolValue(bg))
	return c.PushingNext1(t.Runtime, rt.TableValue(runnerRet)), nil
}

func snailArg(c *rt.GoCont, arg int) (*snail, error) {
	s, ok := valueToSnail(c.Arg(arg))
	if !ok {
		return nil, fmt.Errorf("#%d must be a snail", arg + 1)
	}

	return s, nil
}

func valueToSnail(val rt.Value) (*snail, bool) {
	u, ok := val.TryUserData()
	if !ok {
		return nil, false
	}

	s, ok := u.Value().(*snail)
	return s, ok
}

func snailUserData(s *snail) *rt.UserData {
	snailMeta := s.runtime.Registry(snailMetaKey)
	return rt.NewUserData(s, snailMeta.AsTable())
}
