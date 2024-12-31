package snail

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"hilbish/sink"
	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/lib/iolib"
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
		"run": {srun, 3, false},
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

	streams := &util.Streams{}
	thirdArg := c.Arg(2)
	switch thirdArg.Type() {
		case rt.TableType:
			args := thirdArg.AsTable()

			if luastreams, ok := args.Get(rt.StringValue("sinks")).TryTable(); ok {
				handleStream(luastreams.Get(rt.StringValue("out")), streams, false, false)
				handleStream(luastreams.Get(rt.StringValue("err")), streams, true, false)
				handleStream(luastreams.Get(rt.StringValue("input")), streams, false, true)
			}
		case rt.NilType: // noop
		default:
			return nil, errors.New("expected 3rd arg to either be a table or a boolean")
	}

	var newline bool
	var cont bool
	var luaErr rt.Value = rt.NilValue
	exitCode := 0
	bg, _, _, err := s.Run(cmd, streams)
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
				if exErr, ok := util.IsExecError(err); ok {
					exitCode = exErr.Code
				}
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

func handleStream(v rt.Value, strms *util.Streams, errStream, inStream bool) error {
	if v == rt.NilValue {
		return nil
	}

	ud, ok := v.TryUserData()
	if !ok {
		return errors.New("expected metatable argument")
	}

	val := ud.Value()
	var varstrm io.ReadWriter
	if f, ok := val.(*iolib.File); ok {
		varstrm = f.Handle()
	}

	if f, ok := val.(*sink.Sink); ok {
		varstrm = f.Rw
	}

	if varstrm == nil {
		return errors.New("expected either a sink or file")
	}

	if errStream {
		strms.Stderr = varstrm
	} else if inStream {
		strms.Stdin = varstrm
	} else {
		strms.Stdout = varstrm
	}

	return nil
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
