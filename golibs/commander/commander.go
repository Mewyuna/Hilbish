// library for custom commands
/*
Commander is the library which handles Hilbish commands. This makes
the user able to add Lua-written commands to their shell without making
a separate script in a bin folder. Instead, you may simply use the Commander
library in your Hilbish config.

```lua
local commander = require 'commander'

commander.register('hello', function(args, sinks)
	sinks.out:writeln 'Hello world!'
end)
```

In this example, a command with the name of `hello` is created
that will print `Hello world!` to output. One question you may
have is: What is the `sinks` parameter?

The `sinks` parameter is a table with 3 keys: `input`, `out`, and `err`.
There is an `in` alias to `input`, but it requires using the string accessor syntax (`sinks['in']`)
as `in` is also a Lua keyword, so `input` is preferred for use.
All of them are a @Sink.
In the future, `sinks.in` will be removed.

- `in` is the standard input.
You may use the read functions on this sink to get input from the user.
- `out` is standard output.
This is usually where command output should go.
- `err` is standard error.
This sink is for writing errors, as the name would suggest.
*/
package commander

import (
	"hilbish/moonlight"
	"hilbish/util"
	"hilbish/golibs/bait"
)

type Commander struct{
	Events *bait.Bait
	Commands map[string]*moonlight.Closure
}

func New(rtm *moonlight.Runtime) *Commander {
	c := &Commander{
		Events: bait.New(rtm),
		Commands: make(map[string]*moonlight.Closure),
	}

	return c
}

func (c *Commander) Loader(rtm *moonlight.Runtime) moonlight.Value {
	exports := map[string]moonlight.Export{
		"register": {c.cregister, 2, false},
		"deregister": {c.cderegister, 1, false},
		"registry": {c.cregistry, 0, false},
	}
	mod := moonlight.NewTable()
	rtm.SetExports(mod, exports)

	return moonlight.TableValue(mod)
}

// register(name, cb)
// Adds a new command with the given `name`. When Hilbish has to run a command with a name,
// it will run the function providing the arguments and sinks.
// #param name string Name of the command
// #param cb function Callback to handle command invocation
/*
#example
-- When you run the command `hello` in the shell, it will print `Hello world`.
-- If you run it with, for example, `hello Hilbish`, it will print 'Hello Hilbish'
commander.register('hello', function(args, sinks)
	local name = 'world'
	if #args > 0 then name = args[1] end

	sinks.out:writeln('Hello ' .. name)
end)
#example
*/
func (c *Commander) cregister(mlr *moonlight.Runtime, ct *moonlight.GoCont) (moonlight.Cont, error) {
	cmdName, cmd, err := util.HandleStrCallback(mlr, ct)
	if err != nil {
		return nil, err
	}

	c.Commands[cmdName] = cmd

	return ct.Next(), err
}

// deregister(name)
// Removes the named command. Note that this will only remove Commander-registered commands.
// #param name string Name of the command to remove.
func (c *Commander) cderegister(mlr *moonlight.Runtime, ct *moonlight.GoCont) (moonlight.Cont, error) {
	if err := mlr.Check1Arg(ct); err != nil {
		return nil, err
	}
	cmdName, err := mlr.StringArg(ct, 0)
	if err != nil {
		return nil, err
	}

	delete(c.Commands, cmdName)

	return ct.Next(), err
}

// registry() -> table
// Returns all registered commanders. Returns a list of tables with the following keys:
// - `exec`: The function used to run the commander. Commanders require args and sinks to be passed.
// #returns table
func (c *Commander) cregistry(mlr *moonlight.Runtime, ct *moonlight.GoCont) (moonlight.Cont, error) {
	registryLua := moonlight.NewTable()
	for cmdName, cmd := range c.Commands {
		cmdTbl := moonlight.NewTable()
		//cmdTbl.SetField("exec", moonlight.FunctionValue(cmd))
		print(cmd)
		cmdTbl.SetField("exec", moonlight.StringValue("placeholder"))

		registryLua.SetField(cmdName, moonlight.TableValue(cmdTbl))
	}

	return mlr.PushNext1(ct, moonlight.TableValue(registryLua)), nil
}
