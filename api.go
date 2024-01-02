// the core Hilbish API
// The Hilbish module includes the core API, containing
// interfaces and functions which directly relate to shell functionality.
// #field ver The version of Hilbish
// #field goVersion The version of Go that Hilbish was compiled with
// #field user Username of the user
// #field host Hostname of the machine
// #field dataDir Directory for Hilbish data files, including the docs and default modules
// #field interactive Is Hilbish in an interactive shell?
// #field login Is Hilbish the login shell?
// #field vimMode Current Vim input mode of Hilbish (will be nil if not in Vim input mode)
// #field exitCode Exit code of the last executed command
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/maxlandon/readline"
	"mvdan.cc/sh/v3/interp"
)

var exports = map[string]util.LuaExport{
	"alias": {hlalias, 2, false},
	"appendPath": {hlappendPath, 1, false},
	"complete": {hlcomplete, 2, false},
	"cwd": {hlcwd, 0, false},
	"exec": {hlexec, 1, false},
	"runnerMode": {hlrunnerMode, 1, false},
	"goro": {hlgoro, 1, true},
	"highlighter": {hlhighlighter, 1, false},
	"hinter": {hlhinter, 1, false},
	"multiprompt": {hlmultiprompt, 1, false},
	"prependPath": {hlprependPath, 1, false},
	"prompt": {hlprompt, 1, true},
	"inputMode": {hlinputMode, 1, false},
	"interval": {hlinterval, 2, false},
	"read": {hlread, 1, false},
	"run": {hlrun, 1, true},
	"suspend": {hlsuspend, 0, false},
	"timeout": {hltimeout, 2, false},
	"which": {hlwhich, 1, false},
}

var hshMod *rt.Table
var hilbishLoader = packagelib.Loader{
	Load: hilbishLoad,
	Name: "hilbish",
}

func hilbishLoad(rtm *rt.Runtime) (rt.Value, func()) {
	fakeMod := rt.NewTable()
	modmt := rt.NewTable()
	mod := rt.NewTable()

	modIndex := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		arg := c.Arg(1)
		val := mod.Get(arg)

		return c.PushingNext1(t.Runtime, val), nil
	}
	modNewIndex := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		k, err := c.StringArg(1)
		if err != nil {
			return nil, err
		}

		v := c.Arg(2)
		if k == "highlighter" {
			var err error
			// fine to assign, since itll be either nil or a closure
			highlighter, err = c.ClosureArg(2)
			if err != nil {
				return nil, errors.New("hilbish.highlighter has to be a function")
			}
		} else if k == "hinter" {
			var err error
			hinter, err = c.ClosureArg(2)
			if err != nil {
				return nil, errors.New("hilbish.hinter has to be a function")
			}
		} else if k == "suspend" {
			var err error
			_, err = c.ClosureArg(2)
			if err != nil {
				return nil, errors.New("hilbish.suspend has to be a function")
			}
		} else if modVal := mod.Get(rt.StringValue(k)); modVal != rt.NilValue {
			return nil, errors.New("not allowed to override in hilbish table")
		}
		mod.Set(rt.StringValue(k), v)

		return c.Next(), nil
	}
	modmt.Set(rt.StringValue("__newindex"), rt.FunctionValue(rt.NewGoFunction(modNewIndex, "__newindex", 3, false)))
	modmt.Set(rt.StringValue("__index"), rt.FunctionValue(rt.NewGoFunction(modIndex, "__index", 2, false)))
	fakeMod.SetMetatable(modmt)

	util.SetExports(rtm, mod, exports)
	hshMod = mod

	host, _ := os.Hostname()
	username := curuser.Username

	if runtime.GOOS == "windows" {
		username = strings.Split(username, "\\")[1] // for some reason Username includes the hostname on windows
	}

	util.SetFieldProtected(fakeMod, mod, "ver", rt.StringValue(getVersion()))
	util.SetFieldProtected(fakeMod, mod, "goVersion", rt.StringValue(runtime.Version()))
	util.SetFieldProtected(fakeMod, mod, "user", rt.StringValue(username))
	util.SetFieldProtected(fakeMod, mod, "host", rt.StringValue(host))
	util.SetFieldProtected(fakeMod, mod, "home", rt.StringValue(curuser.HomeDir))
	util.SetFieldProtected(fakeMod, mod, "dataDir", rt.StringValue(dataDir))
	util.SetFieldProtected(fakeMod, mod, "interactive", rt.BoolValue(interactive))
	util.SetFieldProtected(fakeMod, mod, "login", rt.BoolValue(login))
	util.SetFieldProtected(fakeMod, mod, "vimMode", rt.NilValue)
	util.SetFieldProtected(fakeMod, mod, "exitCode", rt.IntValue(0))

	// hilbish.userDir table
	hshuser := userDirLoader(rtm)
	mod.Set(rt.StringValue("userDir"), rt.TableValue(hshuser))

	// hilbish.os table
	hshos := hshosLoader(rtm)
	mod.Set(rt.StringValue("os"), rt.TableValue(hshos))

	// hilbish.aliases table
	aliases = newAliases()
	aliasesModule := aliases.Loader(rtm)
	mod.Set(rt.StringValue("aliases"), rt.TableValue(aliasesModule))

	// hilbish.history table
	historyModule := lr.Loader(rtm)
	mod.Set(rt.StringValue("history"), rt.TableValue(historyModule))

	// hilbish.completion table
	hshcomp := completionLoader(rtm)
	// TODO: REMOVE "completion" AND ONLY USE "completions" WITH AN S
	mod.Set(rt.StringValue("completion"), rt.TableValue(hshcomp))
	mod.Set(rt.StringValue("completions"), rt.TableValue(hshcomp))

	// hilbish.runner table
	runnerModule := runnerModeLoader(rtm)
	mod.Set(rt.StringValue("runner"), rt.TableValue(runnerModule))

	// hilbish.jobs table
	jobs = newJobHandler()
	jobModule := jobs.loader(rtm)
	mod.Set(rt.StringValue("jobs"), rt.TableValue(jobModule))

	// hilbish.timers table
	timers = newTimersModule()
	timersModule := timers.loader(rtm)
	mod.Set(rt.StringValue("timers"), rt.TableValue(timersModule))

	editorModule := editorLoader(rtm)
	mod.Set(rt.StringValue("editor"), rt.TableValue(editorModule))

	versionModule := rt.NewTable()
	util.SetField(rtm, versionModule, "branch", rt.StringValue(gitBranch))
	util.SetField(rtm, versionModule, "full", rt.StringValue(getVersion()))
	util.SetField(rtm, versionModule, "commit", rt.StringValue(gitCommit))
	util.SetField(rtm, versionModule, "release", rt.StringValue(releaseName))
	mod.Set(rt.StringValue("version"), rt.TableValue(versionModule))

	pluginModule := moduleLoader(rtm)
	mod.Set(rt.StringValue("module"), rt.TableValue(pluginModule))

	return rt.TableValue(fakeMod), nil
}

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}

func setVimMode(mode string) {
	util.SetField(l, hshMod, "vimMode", rt.StringValue(mode))
	hooks.Emit("hilbish.vimMode", mode)
}

func unsetVimMode() {
	util.SetField(l, hshMod, "vimMode", rt.NilValue)
}

// run(cmd, returnOut) -> exitCode (number), stdout (string), stderr (string)
// Runs `cmd` in Hilbish's shell script interpreter.
// #param cmd string
// #param returnOut boolean If this is true, the function will return the standard output and error of the command instead of printing it.
// #returns number, string, string
func hlrun(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	cmd, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	var terminalOut bool
	if len(c.Etc()) != 0 {
		tout := c.Etc()[0]
		termOut, ok := tout.TryBool()
		terminalOut = termOut
		if !ok {
			return nil, errors.New("bad argument to run (expected boolean, got " + tout.TypeName() + ")")
		}
	} else {
		terminalOut = true
	}

	var exitcode uint8
	stdout, stderr, err := execCommand(cmd, terminalOut, false)

	if code, ok := interp.IsExitStatus(err); ok {
		exitcode = code
	} else if err != nil {
		exitcode = 1
	}

	stdoutStr := ""
	stderrStr := ""
	if !terminalOut {
		stdoutStr = stdout.(*bytes.Buffer).String()
		stderrStr = stderr.(*bytes.Buffer).String()
	}

	return c.PushingNext(t.Runtime, rt.IntValue(int64(exitcode)), rt.StringValue(stdoutStr), rt.StringValue(stderrStr)), nil
}

// cwd() -> string
// Returns the current directory of the shell.
// #returns string
func hlcwd(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cwd, _ := os.Getwd()

	return c.PushingNext1(t.Runtime, rt.StringValue(cwd)), nil
}


// read(prompt) -> input (string)
// Read input from the user, using Hilbish's line editor/input reader.
// This is a separate instance from the one Hilbish actually uses.
// Returns `input`, will be nil if Ctrl-D is pressed, or an error occurs.
// #param prompt? string Text to print before input, can be empty.
// #returns string|nil
func hlread(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	luaprompt := c.Arg(0)
	if typ := luaprompt.Type(); typ != rt.StringType && typ != rt.NilType {
		return nil, errors.New("expected #1 to be a string")
	}
	prompt, ok := luaprompt.TryString()
	if !ok {
		// if we are here and `luaprompt` is not a string, it's nil
		// substitute with an empty string
		prompt = ""
	}
	
	lualr := &lineReader{
		rl: readline.NewInstance(),
	}
	lualr.SetPrompt(prompt)

	input, err := lualr.Read()
	if err != nil {
		return c.Next(), nil
	}

	return c.PushingNext1(t.Runtime, rt.StringValue(input)), nil
}

/*
prompt(str, typ)
Changes the shell prompt to the provided string.
There are a few verbs that can be used in the prompt text.
These will be formatted and replaced with the appropriate values.
`%d` - Current working directory
`%u` - Name of current user
`%h` - Hostname of device
#param str string
#param typ? string Type of prompt, being left or right. Left by default.
#example
-- the default hilbish prompt without color
hilbish.prompt '%u %d ∆'
-- or something of old:
hilbish.prompt '%u@%h :%d $'
-- prompt: user@hostname: ~/directory $
#example
*/
func hlprompt(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	err := c.Check1Arg()
	if err != nil {
		return nil, err
	}
	p, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	typ := "left"
	// optional 2nd arg
	if len(c.Etc()) != 0 {
		ltyp := c.Etc()[0]
		var ok bool
		typ, ok = ltyp.TryString()
		if !ok {
			return nil, errors.New("bad argument to run (expected string, got " + ltyp.TypeName() + ")")
		}
	}

	switch typ {
		case "left":
			prompt = p
			lr.SetPrompt(fmtPrompt(prompt))
		case "right": lr.SetRightPrompt(fmtPrompt(p))
		default: return nil, errors.New("expected prompt type to be right or left, got " + typ)
	}

	return c.Next(), nil
}

// multiprompt(str)
// Changes the text prompt when Hilbish asks for more input.
// This will show up when text is incomplete, like a missing quote
// #param str string
/*
#example
--[[
imagine this is your text input:
user ~ ∆ echo "hey

but there's a missing quote! hilbish will now prompt you so the terminal
will look like:
user ~ ∆ echo "hey
--> ...!"

so then you get 
user ~ ∆ echo "hey
--> ...!"
hey ...!
]]--
hilbish.multiprompt '-->'
#example
*/
func hlmultiprompt(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	prompt, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	multilinePrompt = prompt

	return c.Next(), nil
}

// alias(cmd, orig)
// Sets an alias, with a name of `cmd` to another command.
// #param cmd string Name of the alias
// #param orig string Command that will be aliased
/*
#example
-- With this, "ga file" will turn into "git add file"
hilbish.alias('ga', 'git add')

-- Numbered substitutions are supported here!
hilbish.alias('dircount', 'ls %1 | wc -l')
-- "dircount ~" would count how many files are in ~ (home directory).
#example
*/
func hlalias(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	cmd, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	orig, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}

	aliases.Add(cmd, orig)

	return c.Next(), nil
}

// appendPath(dir)
// Appends the provided dir to the command path (`$PATH`)
// #param dir string|table Directory (or directories) to append to path
/*
#example
hilbish.appendPath '~/go/bin'
-- Will add ~/go/bin to the command path.

-- Or do multiple:
hilbish.appendPath {
	'~/go/bin',
	'~/.local/bin'
}
#example
*/
func hlappendPath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	arg := c.Arg(0)

	// check if dir is a table or a string
	if arg.Type() == rt.TableType {
		util.ForEach(arg.AsTable(), func(k rt.Value, v rt.Value) {
			if v.Type() == rt.StringType {
				appendPath(v.AsString())
			}
		})
	} else if arg.Type() == rt.StringType {
		appendPath(arg.AsString())
	} else {
		return nil, errors.New("bad argument to appendPath (expected string or table, got " + arg.TypeName() + ")")
	}

	return c.Next(), nil
}

func appendPath(dir string) {
	dir = strings.Replace(dir, "~", curuser.HomeDir, 1)
	pathenv := os.Getenv("PATH")

	// if dir isnt already in $PATH, add it
	if !strings.Contains(pathenv, dir) {
		os.Setenv("PATH", pathenv + string(os.PathListSeparator) + dir)
	}
}

// exec(cmd)
// Replaces the currently running Hilbish instance with the supplied command.
// This can be used to do an in-place restart.
// #param cmd string
func hlexec(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	cmd, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	cmdArgs, _ := splitInput(cmd)
	if runtime.GOOS != "windows" {
		cmdPath, err := exec.LookPath(cmdArgs[0])
		if err != nil {
			fmt.Println(err)
			// if we get here, cmdPath will be nothing
			// therefore nothing will run
		}

		// syscall.Exec requires an absolute path to a binary
		// path, args, string slice of environments
		syscall.Exec(cmdPath, cmdArgs, os.Environ())
	} else {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
		os.Exit(0)
	}

	return c.Next(), nil
}

// goro(fn)
// Puts `fn` in a Goroutine.
// This can be used to run any function in another thread at the same time as other Lua code.
// **NOTE: THIS FUNCTION MAY CRASH HILBISH IF OUTSIDE VARIABLES ARE ACCESSED.**
// **This is a limitation of the Lua runtime.**
// #param fn function
func hlgoro(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	fn, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}

	// call fn
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// do something here?
			}
		}()

		_, err := rt.Call1(l.MainThread(), rt.FunctionValue(fn), c.Etc()...)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error in goro function:\n\n", err)
		}
	}()

	return c.Next(), nil
}

// timeout(cb, time) -> @Timer
// Executed the `cb` function after a period of `time`.
// This creates a Timer that starts ticking immediately.
// #param cb function
// #param time number Time to run in milliseconds.
// #returns Timer
func hltimeout(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	cb, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}
	ms, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}

	interval := time.Duration(ms) * time.Millisecond
	timer := timers.create(timerTimeout, interval, cb)
	timer.start()
	
	return c.PushingNext1(t.Runtime, rt.UserDataValue(timer.ud)), nil
}

// interval(cb, time) -> @Timer
// Runs the `cb` function every specified amount of `time`.
// This creates a timer that ticking immediately.
// #param cb function
// #param time number Time in milliseconds.
// #return Timer
func hlinterval(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	cb, err := c.ClosureArg(0)
	if err != nil {
		return nil, err
	}
	ms, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}

	interval := time.Duration(ms) * time.Millisecond
	timer := timers.create(timerInterval, interval, cb)
	timer.start()

	return c.PushingNext1(t.Runtime, rt.UserDataValue(timer.ud)), nil
}

// complete(scope, cb)
// Registers a completion handler for the specified scope.
// A `scope` is expected to be `command.<cmd>`,
// replacing <cmd> with the name of the command (for example `command.git`).
// The documentation for completions, under Features/Completions or `doc completions`
// provides more details.
// #param scope string
// #param cb function
/*
#example
-- This is a very simple example. Read the full doc for completions for details.
hilbish.complete('command.sudo', function(query, ctx, fields)
	if #fields == 0 then
		-- complete for commands
		local comps, pfx = hilbish.completion.bins(query, ctx, fields)
		local compGroup = {
			items = comps, -- our list of items to complete
			type = 'grid' -- what our completions will look like.
		}

		return {compGroup}, pfx
	end

	-- otherwise just be boring and return files

	local comps, pfx = hilbish.completion.files(query, ctx, fields)
	local compGroup = {
		items = comps,
		type = 'grid'
	}

	return {compGroup}, pfx
end)
#example
*/
func hlcomplete(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	scope, cb, err := util.HandleStrCallback(t, c)
	if err != nil {
		return nil, err
	}
	luaCompletions[scope] = cb

	return c.Next(), nil
}

// prependPath(dir)
// Prepends `dir` to $PATH.
// #param dir string
func hlprependPath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	dir, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	dir = strings.Replace(dir, "~", curuser.HomeDir, 1)
	pathenv := os.Getenv("PATH")

	// if dir isnt already in $PATH, add in
	if !strings.Contains(pathenv, dir) {
		os.Setenv("PATH", dir + string(os.PathListSeparator) + pathenv)
	}

	return c.Next(), nil
}

// which(name) -> string
// Checks if `name` is a valid command.
// Will return the path of the binary, or a basename if it's a commander.
// #param name string
// #returns string
func hlwhich(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	name, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	// itll return either the original command or what was passed
	// if name isnt empty its not an issue
	alias := aliases.Resolve(name)
	cmd := strings.Split(alias, " ")[0]

	// check for commander
	if commands[cmd] != nil {
		// they dont resolve to a path, so just send the cmd
		return c.PushingNext1(t.Runtime, rt.StringValue(cmd)), nil
	}

	path, err := exec.LookPath(cmd)
	if err != nil {
		return c.Next(), nil
	}

	return c.PushingNext1(t.Runtime, rt.StringValue(path)), nil
}

// inputMode(mode)
// Sets the input mode for Hilbish's line reader.
// `emacs` is the default. Setting it to `vim` changes behavior of input to be
// Vim-like with modes and Vim keybinds.
// #param mode string Can be set to either `emacs` or `vim`
func hlinputMode(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	mode, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	switch mode {
		case "emacs":
			unsetVimMode()
			lr.rl.InputMode = readline.Emacs
		case "vim":
			setVimMode("insert")
			lr.rl.InputMode = readline.Vim
		default:
			return nil, errors.New("inputMode: expected vim or emacs, received " + mode)
	}

	return c.Next(), nil
}

// runnerMode(mode)
// Sets the execution/runner mode for interactive Hilbish.
// This determines whether Hilbish wll try to run input as Lua
// and/or sh or only do one of either.
// Accepted values for mode are hybrid (the default), hybridRev (sh first then Lua),
// sh, and lua. It also accepts a function, to which if it is passed one
// will call it to execute user input instead.
// Read [about runner mode](../features/runner-mode) for more information.
// #param mode string|function
func hlrunnerMode(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	mode := c.Arg(0)

	switch mode.Type() {
		case rt.StringType:
			switch mode.AsString() {
				case "hybrid", "hybridRev", "lua", "sh": runnerMode = mode
				default: return nil, errors.New("execMode: expected either a function or hybrid, hybridRev, lua, sh. Received " + mode.AsString())
			}
		case rt.FunctionType: runnerMode = mode
		default: return nil, errors.New("execMode: expected either a function or hybrid, hybridRev, lua, sh. Received " + mode.TypeName())
	}

	return c.Next(), nil
}

// hinter(line, pos)
// The command line hint handler. It gets called on every key insert to
// determine what text to use as an inline hint. It is passed the current
// line and cursor position. It is expected to return a string which is used
// as the text for the hint. This is by default a shim. To set hints,
// override this function with your custom handler.
// #param line string
// #param pos number Position of cursor in line. Usually equals string.len(line)
/*
#example
-- this will display "hi" after the cursor in a dimmed color.
function hilbish.hinter(line, pos)
	return 'hi'
end
#example
*/
func hlhinter(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	return c.Next(), nil
}

// highlighter(line)
// Line highlighter handler.
// This is mainly for syntax highlighting, but in reality could set the input
// of the prompt to *display* anything. The callback is passed the current line
// and is expected to return a line that will be used as the input display.
// Note that to set a highlighter, one has to override this function.
// #example
// --This code will highlight all double quoted strings in green.
// function hilbish.highlighter(line)
//    return line:gsub('"%w+"', function(c) return lunacolors.green(c) end)
// end
// #example
// #param line string
func hlhighlighter(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	return c.Next(), nil
}

// suspend()
// Suspends the currently running process. This can (and should be) overwritten if a
// a different runner is being used.
// --- @param line string
func hlsuspend(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if currentHandle != nil || currentHandle.Process != nil {
		j := jobs.addFromHandler(currentCmd, currentHandle)
		j.suspend()
	}

	return c.Next(), nil
}
