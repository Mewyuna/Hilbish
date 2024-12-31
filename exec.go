package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"hilbish/util"
	//herror "hilbish/errors"

	rt "github.com/arnodel/golua/runtime"
	//"github.com/yuin/gopher-lua/parse"
)

var errNotExec = errors.New("not executable")
var errNotFound = errors.New("not found")
var runnerMode rt.Value = rt.NilValue

func runInput(input string, priv bool) {
	running = true
	cmdString := aliases.Resolve(input)
	hooks.Emit("command.preexec", input, cmdString)

	currentRunner := runnerMode

	rerun:
	var exitCode uint8
	var cont bool
	var newline bool
	// save incase it changes while prompting (For some reason)
	input, exitCode, cont, newline, runnerErr, err := runLuaRunner(currentRunner, input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		cmdFinish(124, input, priv)
		return
	}
	// we only use `err` to check for lua eval error
	// our actual error should only be a runner provided error at this point
	// command not found type, etc
	err = runnerErr

	if cont {
		input, err = continuePrompt(input, newline)
		if err == nil {
			goto rerun
		} else if err == io.EOF {
			lr.SetPrompt(fmtPrompt(prompt))
		}
	}

	if err != nil && err != io.EOF {
		if exErr, ok := util.IsExecError(err); ok {
			hooks.Emit("command." + exErr.Typ, exErr.Cmd)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	cmdFinish(exitCode, input, priv)
}

func reprompt(input string, newline bool) (string, error) {
	for {
		/*
		if strings.HasSuffix(input, "\\") {
			input = strings.TrimSuffix(input, "\\") + "\n"
		}
		*/
		in, err := continuePrompt(input, newline)
		if err != nil {
			lr.SetPrompt(fmtPrompt(prompt))
			return input, err
		}

		return in, nil
	}
}

func runLuaRunner(runr rt.Value, userInput string) (input string, exitCode uint8, continued bool, newline bool, runnerErr, err error) {
	term := rt.NewTerminationWith(l.MainThread().CurrentCont(), 3, false)
	err = rt.Call(l.MainThread(), runr, []rt.Value{rt.StringValue(userInput)}, term)
	if err != nil {
		return "", 124, false, false, nil, err
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

	if nl, ok := runner.Get(rt.StringValue("newline")).TryBool(); ok {
		newline = nl
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

/*
func execSh(cmdString string) (input string, exitcode uint8, cont bool, newline bool, e error) {
	_, _, err := execCommand(cmdString, nil)
	if err != nil {
		// If input is incomplete, start multiline prompting
		if syntax.IsIncomplete(err) {
			if !interactive {
				return cmdString, 126, false, false, err
			}

			newline := false
			if strings.Contains(err.Error(), "unclosed here-document") {
				newline = true
			}
			return cmdString, 126, true, newline, err
		} else {
			if code, ok := interp.IsExitStatus(err); ok {
				return cmdString, code, false, false, nil
			} else {
				return cmdString, 126, false, false, err
			}
		}
	}

	return cmdString, 0, false, false, nil
}
*/

func splitInput(input string) ([]string, string) {
	// end my suffering
	// TODO: refactor this garbage
	quoted := false
	cmdArgs := []string{}
	sb := &strings.Builder{}
	cmdstr := &strings.Builder{}

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
