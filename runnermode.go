package main

import (
	"hilbish/moonlight"
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

- `exitCode` (number): A numerical code to indicate the exit result.
- `input` (string): The user input. This will be used to add
to the history.
- `err` (string): A string to indicate an interal error for the runner.
It can be set to a few special values for Hilbish to throw the right hooks and have a better looking message:

`[command]: not-found` will throw a command.not-found hook based on what `[command]` is.  

`[command]: not-executable` will throw a command.not-executable hook.
- `continue` (boolean): Whether to prompt the user for more input.

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
func runnerModeLoader(rtm *moonlight.Runtime) *moonlight.Table {
	exports := map[string]moonlight.Export{
		"sh": {shRunner, 1, false},
		"lua": {luaRunner, 1, false},
		"setMode": {hlrunnerMode, 1, false},
	}

	mod := moonlight.NewTable()
	rtm.SetExports(mod, exports)

	return mod
}

// #interface runner
// setMode(cb)
// This is the same as the `hilbish.runnerMode` function.
// It takes a callback, which will be used to execute all interactive input.
// In normal cases, neither callbacks should be overrided by the user,
// as the higher level functions listed below this will handle it.
// #param cb function
func _runnerMode() {}

// #interface runner
// sh(cmd)
// Runs a command in Hilbish's shell script interpreter.
// This is the equivalent of using `source`.
// #param cmd string
func shRunner(mlr *moonlight.Runtime, c *moonlight.GoCont) (moonlight.Cont, error) {
	if err := mlr.Check1Arg(c); err != nil {
		return nil, err
	}
	cmd, err := mlr.StringArg(c, 0)
	if err != nil {
		return nil, err
	}

	_, exitCode, cont, err := execSh(aliases.Resolve(cmd))
	var luaErr moonlight.Value = moonlight.NilValue
	if err != nil {
		luaErr = moonlight.StringValue(err.Error())
	}
	runnerRet := moonlight.NewTable()
	runnerRet.SetField("input", moonlight.StringValue(cmd))
	runnerRet.SetField("exitCode", moonlight.IntValue(int64(exitCode)))
	runnerRet.SetField("continue", moonlight.BoolValue(cont))
	runnerRet.SetField("err", luaErr)

	return mlr.PushNext1(c, moonlight.TableValue(runnerRet)), nil
}

// #interface runner
// lua(cmd)
// Evaluates `cmd` as Lua input. This is the same as using `dofile`
// or `load`, but is appropriated for the runner interface.
// #param cmd string
func luaRunner(mlr *moonlight.Runtime, c *moonlight.GoCont) (moonlight.Cont, error) {
	if err := mlr.Check1Arg(c); err != nil {
		return nil, err
	}
	cmd, err := mlr.StringArg(c, 0)
	if err != nil {
		return nil, err
	}

	input, exitCode, err := handleLua(cmd)
	var luaErr moonlight.Value = moonlight.NilValue
	if err != nil {
		luaErr = moonlight.StringValue(err.Error())
	}
	runnerRet := moonlight.NewTable()
	runnerRet.SetField("input", moonlight.StringValue(input))
	runnerRet.SetField("exitCode", moonlight.IntValue(int64(exitCode)))
	runnerRet.SetField("err", luaErr)


	return mlr.PushNext1(c, moonlight.TableValue(runnerRet)), nil
}
