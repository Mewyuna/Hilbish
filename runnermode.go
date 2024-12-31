package main

import (
	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
)

// #interface runner
// interactive command runner customization
/* The runner interface contains functions that allow the user to change
how Hilbish interprets interactive input.
Users can add and change the default runner for interactive input to any
language or script of their choosing. A good example is using it to
write command in Fennel.

Runners are functions that evaluate user input. The default runners in
Hilbish can run shell script and Lua code.

A runner is passed the input and has to return a table with these values.
All are not required, only the useful ones the runner needs to return.
(So if there isn't an error, just omit `err`.)

- `exitCode` (number): Exit code of the command
- `input` (string): The text input of the user. This is used by Hilbish to append extra input, in case
more is requested.
- `err` (string): A string that represents an error from the runner.
This should only be set when, for example, there is a syntax error.
It can be set to a few special values for Hilbish to throw the right
hooks and have a better looking message.
	- `<command>: not-found` will throw a `command.not-found` hook
	based on what `<command>` is.
	- `<command>: not-executable` will throw a `command.not-executable` hook.
- `continue` (boolean): Whether Hilbish should prompt the user for no input
- `newline` (boolean): Whether a newline should be added at the end of `input`.

Here is a simple example of a fennel runner. It falls back to
shell script if fennel eval has an error.
```lua
local fennel = require 'fennel'

hilbish.runnerMode(function(input)
	local ok = pcall(fennel.eval, input)
	if ok then
		return {
			input = input
		}
	end

	return hilbish.runner.sh(input)
end)
```
*/
func runnerModeLoader(rtm *rt.Runtime) *rt.Table {
	exports := map[string]util.LuaExport{
		//"sh": {shRunner, 1, false},
		"lua": {luaRunner, 1, false},
		"setMode": {hlrunnerMode, 1, false},
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, exports)

	return mod
}

// #interface runner
// setMode(cb)
// **NOTE: This function is deprecated and will be removed in 3.0**
// Use `hilbish.runner.setCurrent` instead.
// This is the same as the `hilbish.runnerMode` function.
// It takes a callback, which will be used to execute all interactive input.
// In normal cases, neither callbacks should be overrided by the user,
// as the higher level functions (setCurrent) this will handle it.
// #param cb function
func _runnerMode() {}

// #interface runner
// sh(cmd)
// Runs a command in Hilbish's shell script interpreter.
// This is the equivalent of using `source`.
// #param cmd string
/*
func shRunner(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	cmd, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	_, exitCode, cont, newline, err := execSh(aliases.Resolve(cmd))
	var luaErr rt.Value = rt.NilValue
	if err != nil {
		luaErr = rt.StringValue(err.Error())
	}
	runnerRet := rt.NewTable()
	runnerRet.Set(rt.StringValue("input"), rt.StringValue(cmd))
	runnerRet.Set(rt.StringValue("exitCode"), rt.IntValue(int64(exitCode)))
	runnerRet.Set(rt.StringValue("continue"), rt.BoolValue(cont))
	runnerRet.Set(rt.StringValue("newline"), rt.BoolValue(newline))
	runnerRet.Set(rt.StringValue("err"), luaErr)

	return c.PushingNext(t.Runtime, rt.TableValue(runnerRet)), nil
}
*/

// #interface runner
// lua(cmd)
// Evaluates `cmd` as Lua input. This is the same as using `dofile`
// or `load`, but is appropriated for the runner interface.
// #param cmd string
func luaRunner(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	cmd, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	input, exitCode, err := handleLua(cmd)
	var luaErr rt.Value = rt.NilValue
	if err != nil {
		luaErr = rt.StringValue(err.Error())
	}
	runnerRet := rt.NewTable()
	runnerRet.Set(rt.StringValue("input"), rt.StringValue(input))
	runnerRet.Set(rt.StringValue("exitCode"), rt.IntValue(int64(exitCode)))
	runnerRet.Set(rt.StringValue("err"), luaErr)

	return c.PushingNext(t.Runtime, rt.TableValue(runnerRet)), nil
}
