package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
	"mvdan.cc/sh/v3/shell"
	//"github.com/yuin/gopher-lua/parse"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"mvdan.cc/sh/v3/expand"
)

var errNotExec = errors.New("not executable")
var errNotFound = errors.New("not found")
var runnerMode rt.Value = rt.StringValue("hybrid")
var currentHandle *exec.Cmd
var currentCmd string

type execError struct{
	typ string
	cmd string
	code int
	colon bool
	err error
}

func (e execError) Error() string {
	return fmt.Sprintf("%s: %s", e.cmd, e.typ)
}

func (e execError) sprint() error {
	sep := " "
	if e.colon {
		sep = ": "
	}

	return fmt.Errorf("hilbish: %s%s%s", e.cmd, sep, e.err.Error())
}

func isExecError(err error) (execError, bool) {
	if exErr, ok := err.(execError); ok {
		return exErr, true
	}

	fields := strings.Split(err.Error(), ": ")
	knownTypes := []string{
		"not-found",
		"not-executable",
	}

	if len(fields) > 1 && contains(knownTypes, fields[1]) {
		var colon bool
		var e error
		switch fields[1] {
			case "not-found":
				e = errNotFound
			case "not-executable":
				colon = true
				e = errNotExec
		}

		return execError{
			cmd: fields[0],
			typ: fields[1],
			colon: colon,
			err: e,
		}, true
	}

	return execError{}, false
}

func runInput(input string, priv bool) {
	running = true
	cmdString := aliases.Resolve(input)
	hooks.Emit("command.preexec", input, cmdString)

	rerun:
	var exitCode uint8
	var err error
	var cont bool
	// save incase it changes while prompting (For some reason)
	currentRunner := runnerMode
	if currentRunner.Type() == rt.StringType {
		switch currentRunner.AsString() {
			case "hybrid":
				_, _, err = handleLua(input)
				if err == nil {
					cmdFinish(0, input, priv)
					return
				}
				input, exitCode, cont, err = handleSh(input)
			case "hybridRev":
				_, _, _, err = handleSh(input)
				if err == nil {
					cmdFinish(0, input, priv)
					return
				}
				input, exitCode, err = handleLua(input)
			case "lua":
				input, exitCode, err = handleLua(input)
			case "sh":
				input, exitCode, cont, err = handleSh(input)
		}
	} else {
		// can only be a string or function so
		var runnerErr error
		input, exitCode, cont, runnerErr, err = runLuaRunner(currentRunner, input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			cmdFinish(124, input, priv)
			return
		}
		// yep, we only use `err` to check for lua eval error
		// our actual error should only be a runner provided error at this point
		// command not found type, etc
		err = runnerErr
	}

	if cont {
		input, err = reprompt(input)
		if err == nil {
			goto rerun
		} else if err == io.EOF {
			return
		}
	}

	if err != nil {
		if exErr, ok := isExecError(err); ok {
			hooks.Emit("command." + exErr.typ, exErr.cmd)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	cmdFinish(exitCode, input, priv)
}

func reprompt(input string) (string, error) {
	for {
		in, err := continuePrompt(strings.TrimSuffix(input, "\\"))
		if err != nil {
			lr.SetPrompt(fmtPrompt(prompt))
			return input, err
		}

		if strings.HasSuffix(in, "\\") {
			continue
		}
		return in, nil
	}
}

func runLuaRunner(runr rt.Value, userInput string) (input string, exitCode uint8, continued bool, runnerErr, err error) {
	term := rt.NewTerminationWith(l.MainThread().CurrentCont(), 3, false)
	err = rt.Call(l.MainThread(), runr, []rt.Value{rt.StringValue(userInput)}, term)
	if err != nil {
		return "", 124, false, nil, err
	}

	var runner *rt.Table
	var ok bool
	runnerRet := term.Get(0)
	if runner, ok = runnerRet.TryTable(); !ok {
		fmt.Fprintln(os.Stderr, "runner did not return a table")
		exitCode = 125
		input = userInput
		return
	}

	if code, ok := runner.Get(rt.StringValue("exitCode")).TryInt(); ok {
		exitCode = uint8(code)
	}

	if inp, ok := runner.Get(rt.StringValue("input")).TryString(); ok {
		input = inp
	}

	if errStr, ok := runner.Get(rt.StringValue("err")).TryString(); ok {
		runnerErr = fmt.Errorf("%s", errStr)
	}

	if c, ok := runner.Get(rt.StringValue("continue")).TryBool(); ok {
		continued = c
	}
	return
}

func handleLua(input string) (string, uint8, error) {
	cmdString := aliases.Resolve(input)
	// First try to load input, essentially compiling to bytecode
	chunk, err := l.CompileAndLoadLuaChunk("", []byte(cmdString), rt.TableValue(l.GlobalEnv()))
	if err != nil && noexecute {
		fmt.Println(err)
	/*	if lerr, ok := err.(*lua.ApiError); ok {
			if perr, ok := lerr.Cause.(*parse.Error); ok {
				print(perr.Pos.Line == parse.EOF)
			}
		}
	*/
		return cmdString, 125, err
	}
	// And if there's no syntax errors and -n isnt provided, run
	if !noexecute {
		if chunk != nil {
			_, err = rt.Call1(l.MainThread(), rt.FunctionValue(chunk))
		}
	}
	if err == nil {
		return cmdString, 0, nil
	}

	return cmdString, 125, err
}

func handleSh(cmdString string) (input string, exitCode uint8, cont bool, runErr error) {
	shRunner := hshMod.Get(rt.StringValue("runner")).AsTable().Get(rt.StringValue("sh"))
	var err error
	input, exitCode, cont, runErr, err = runLuaRunner(shRunner, cmdString)
	if err != nil {
		runErr = err
	}
	return
}

func execSh(cmdString string) (string, uint8, bool, error) {
	_, _, err := execCommand(cmdString, true, true)
	if err != nil {
		// If input is incomplete, start multiline prompting
		if syntax.IsIncomplete(err) {
			if !interactive {
				return cmdString, 126, false, err
			}
			return cmdString, 126, true, err
		} else {
			if code, ok := interp.IsExitStatus(err); ok {
				return cmdString, code, false, nil
			} else {
				return cmdString, 126, false, err
			}
		}
	}

	return cmdString, 0, false, nil
}

// Run command in sh interpreter
func execCommand(cmd string, terminalOut, interactiveCall bool) (io.Writer, io.Writer, error) {
	file, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		return nil, nil, err
	}

	runner, _ := interp.New()

	var stdout io.Writer
	var stderr io.Writer
	if terminalOut {
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr)(runner)
	} else {
		stdout = new(bytes.Buffer)
		stderr = new(bytes.Buffer)
		interp.StdIO(os.Stdin, stdout, stderr)(runner)
	}
	buf := new(bytes.Buffer)
	printer := syntax.NewPrinter()

	var bg bool
	for _, stmt := range file.Stmts {
		bg = false
		printer.Print(buf, stmt.Cmd)

		stmtStr := buf.String()
		buf.Reset()

		if interactiveCall {
			currentCmd = stmtStr
		}

		if stmt.Background {
			bg = true
			jobs.add(stmtStr, []string{}, "")
		}

		interp.ExecHandler(execHandle(bg, interactiveCall))(runner)
		err = runner.Run(context.TODO(), stmt)
		if err != nil {
			return stdout, stderr, err
		}
	}

	return stdout, stderr, nil
}

func execHandle(bg, interactiveCall bool) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		_, argstring := splitInput(strings.Join(args, " "))
		// i dont really like this but it works
		if aliases.All()[args[0]] != "" {
			for i, arg := range args {
				if strings.Contains(arg, " ") {
					args[i] = fmt.Sprintf("\"%s\"", arg)
				}
			}
			_, argstring = splitInput(strings.Join(args, " "))

			// If alias was found, use command alias
			argstring = aliases.Resolve(argstring)
			var err error
			args, err = shell.Fields(argstring, nil)
			if err != nil {
				return err
			}
		}

		// If command is defined in Lua then run it
		luacmdArgs := rt.NewTable()
		for i, str := range args[1:] {
			luacmdArgs.Set(rt.IntValue(int64(i + 1)), rt.StringValue(str))
		}

		hc := interp.HandlerCtx(ctx)
		if commands[args[0]] != nil {
			stdin := newSinkInput(hc.Stdin)
			stdout := newSinkOutput(hc.Stdout)
			stderr := newSinkOutput(hc.Stderr)

			sinks := rt.NewTable()
			sinks.Set(rt.StringValue("in"), rt.UserDataValue(stdin.ud))
			sinks.Set(rt.StringValue("out"), rt.UserDataValue(stdout.ud))
			sinks.Set(rt.StringValue("err"), rt.UserDataValue(stderr.ud))

			luaexitcode, err := rt.Call1(l.MainThread(), rt.FunctionValue(commands[args[0]]), rt.TableValue(luacmdArgs), rt.TableValue(sinks))
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error in command:\n" + err.Error())
				return interp.NewExitStatus(1)
			}

			var exitcode uint8

			if code, ok := luaexitcode.TryInt(); ok {
				exitcode = uint8(code)
			} else if luaexitcode != rt.NilValue {
				// deregister commander
				delete(commands, args[0])
				fmt.Fprintf(os.Stderr, "Commander did not return number for exit code. %s, you're fired.\n", args[0])
			}

			return interp.NewExitStatus(exitcode)
		}

		err := lookpath(args[0])
		if err == errNotExec {
			return execError{
				typ: "not-executable",
				cmd: args[0],
				code: 126,
				colon: true,
				err: errNotExec,
			}
		} else if err != nil {
			return execError{
				typ: "not-found",
				cmd: args[0],
				code: 127,
				err: errNotFound,
			}
		}

		killTimeout := 2 * time.Second
		// from here is basically copy-paste of the default exec handler from
		// sh/interp but with our job handling
		path, err := interp.LookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			fmt.Fprintln(hc.Stderr, err)
			return interp.NewExitStatus(127)
		}

		env := hc.Env
		envList := make([]string, 0, 64)
		env.Each(func(name string, vr expand.Variable) bool {
			if !vr.IsSet() {
				// If a variable is set globally but unset in the
				// runner, we need to ensure it's not part of the final
				// list. Seems like zeroing the element is enough.
				// This is a linear search, but this scenario should be
				// rare, and the number of variables shouldn't be large.
				for i, kv := range envList {
					if strings.HasPrefix(kv, name+"=") {
						envList[i] = ""
					}
				}
			}
			if vr.Exported && vr.Kind == expand.String {
				envList = append(envList, name+"="+vr.String())
			}
			return true
		})
		cmd := exec.Cmd{
			Path: path,
			Args: args,
			Env: envList,
			Dir: hc.Dir,
			Stdin: hc.Stdin,
			Stdout: hc.Stdout,
			Stderr: hc.Stderr,
		}

		if interactiveCall {
			currentHandle = &cmd
		}

		var j *job
		if bg {
			j = jobs.getLatest()
			j.setHandle(&cmd)
			err = j.start()
		} else {
			err = cmd.Start()
		}

		execDone := make(chan struct{}, 1)
		var suspended bool
		if err == nil {
			if done := ctx.Done(); done != nil {
				go func() {
					<-done

					if killTimeout <= 0 || runtime.GOOS == "windows" {
						cmd.Process.Signal(os.Kill)
						return
					}

					// TODO: don't temporarily leak this goroutine
					// if the program stops itself with the
					// interrupt.
					go func() {
						time.Sleep(killTimeout)
						cmd.Process.Signal(os.Kill)
					}()
					cmd.Process.Signal(os.Interrupt)
				}()
			}

			go func() {
				hooks.On("job.suspend", func(args ...interface{}) {
					fmt.Println("suspended")
					val := args[0].(rt.Value)

					_, ok := valueToJob(val)
					if !ok {
						return
					}

					suspended = true
					execDone <- struct{}{}
				})
			}()
			go func() {
				err = cmd.Wait()
				execDone <- struct{}{}
			}()

			<-execDone
		}

		exit := handleExecErr(err)

		if bg && !suspended {
			j.exitCode = int(exit)
			j.finish()
		}
		return interp.NewExitStatus(exit)
	}
}

func handleExecErr(err error) (exit uint8) {
	ctx := context.TODO()

	switch x := err.(type) {
	case *exec.ExitError:
		// started, but errored - default to 1 if OS
		// doesn't have exit statuses
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				if ctx.Err() != nil {
					return
				}
				exit = uint8(128 + status.Signal())
				return
			}
			exit = uint8(status.ExitStatus())
			return
		}
		exit = 1
		return
	case *exec.Error:
		// did not start
		//fmt.Fprintf(hc.Stderr, "%v\n", err)
		exit = 127
	default: return
	}

	return
}
func lookpath(file string) error { // custom lookpath function so we know if a command is found *and* is executable
	var skip []string
	if runtime.GOOS == "windows" {
		skip = []string{"./", "../", "~/", "C:"}
	} else {
		skip = []string{"./", "/", "../", "~/"}
	}
	for _, s := range skip {
		if strings.HasPrefix(file, s) {
			return findExecutable(file, false, false)
		}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		path := filepath.Join(dir, file)
		err := findExecutable(path, true, false)
		if err == errNotExec {
			return err
		} else if err == nil {
			return nil
		}
	}

	return os.ErrNotExist
}

func splitInput(input string) ([]string, string) {
	// end my suffering
	// TODO: refactor this garbage
	quoted := false
	startlastcmd := false
	lastcmddone := false
	cmdArgs := []string{}
	sb := &strings.Builder{}
	cmdstr := &strings.Builder{}
	lastcmd := "" //readline.GetHistory(readline.HistorySize() - 1)

	for _, r := range input {
		if r == '"' {
			// start quoted input
			// this determines if other runes are replaced
			quoted = !quoted
			// dont add back quotes
			//sb.WriteRune(r)
		} else if !quoted && r == '~' {
			// if not in quotes and ~ is found then make it $HOME
			sb.WriteString(os.Getenv("HOME"))
		} else if !quoted && r == ' ' {
			// if not quoted and there's a space then add to cmdargs
			cmdArgs = append(cmdArgs, sb.String())
			sb.Reset()
		} else if !quoted && r == '^' && startlastcmd && !lastcmddone {
			// if ^ is found, isnt in quotes and is
			// the second occurence of the character and is
			// the first time "^^" has been used
			cmdstr.WriteString(lastcmd)
			sb.WriteString(lastcmd)

			startlastcmd = !startlastcmd
			lastcmddone = !lastcmddone

			continue
		} else if !quoted && r == '^' && !lastcmddone {
			// if ^ is found, isnt in quotes and is the
			// first time of starting "^^"
			startlastcmd = !startlastcmd
			continue
		} else {
			sb.WriteRune(r)
		}
		cmdstr.WriteRune(r)
	}
	if sb.Len() > 0 {
		cmdArgs = append(cmdArgs, sb.String())
	}

	return cmdArgs, cmdstr.String()
}

func cmdFinish(code uint8, cmdstr string, private bool) {
	util.SetField(l, hshMod, "exitCode", rt.IntValue(int64(code)))
	// using AsValue (to convert to lua type) on an interface which is an int
	// results in it being unknown in lua .... ????
	// so we allow the hook handler to take lua runtime Values
	hooks.Emit("command.exit", rt.IntValue(int64(code)), cmdstr, private)
}
