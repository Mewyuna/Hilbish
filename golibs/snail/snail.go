// shell script interpreter library
package snail

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"hilbish/sink"
	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
	"mvdan.cc/sh/v3/shell"
	//"github.com/yuin/gopher-lua/parse"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"mvdan.cc/sh/v3/expand"
)

type snail struct{
	runner *interp.Runner
	runtime *rt.Runtime
}

func New(rtm *rt.Runtime) *snail {
	runner, _ := interp.New()

	return &snail{
		runner: runner,
		runtime: rtm,
	}
}

func (s *snail) Run(cmd string, strms *util.Streams) (bool, io.Writer, io.Writer, error){
	file, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		return false, nil, nil, err
	}

	if strms == nil {
		strms = &util.Streams{}
	}

	if strms.Stdout == nil {
		strms.Stdout = os.Stdout
	}

	if strms.Stderr == nil {
		strms.Stderr = os.Stderr
	}

	if strms.Stdin == nil {
		strms.Stdin = os.Stdin
	}

	interp.StdIO(strms.Stdin, strms.Stdout, strms.Stderr)(s.runner)
	interp.Env(nil)(s.runner)

	buf := new(bytes.Buffer)
	//printer := syntax.NewPrinter()

	var bg bool
	for _, stmt := range file.Stmts {
		bg = false
		if stmt.Background {
			bg = true
			//printer.Print(buf, stmt.Cmd)

			//stmtStr := buf.String()
			buf.Reset()
			//jobs.add(stmtStr, []string{}, "")
		}

		interp.ExecHandler(func(ctx context.Context, args []string) error {
			_, argstring := splitInput(strings.Join(args, " "))
			// i dont really like this but it works
			aliases := make(map[string]string)
			aliasesLua, _ := util.DoString(s.runtime, "return hilbish.aliases.list()")
			util.ForEach(aliasesLua.AsTable(), func(k, v rt.Value) {
				aliases[k.AsString()] = v.AsString()
			})
			if aliases[args[0]] != "" {
				for i, arg := range args {
					if strings.Contains(arg, " ") {
						args[i] = fmt.Sprintf("\"%s\"", arg)
					}
				}
				_, argstring = splitInput(strings.Join(args, " "))

				// If alias was found, use command alias
				argstring = util.MustDoString(s.runtime, fmt.Sprintf(`return hilbish.aliases.resolve("%s")`, argstring)).AsString()

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

			cmds := make(map[string]*rt.Closure)
			luaCmds := util.MustDoString(s.runtime, "local commander = require 'commander'; return commander.registry()").AsTable()
			util.ForEach(luaCmds, func(k, v rt.Value) {
				cmds[k.AsString()] = v.AsTable().Get(rt.StringValue("exec")).AsClosure()
			})
			if cmd := cmds[args[0]]; cmd != nil {
				stdin := sink.NewSinkInput(s.runtime, hc.Stdin)
				stdout := sink.NewSinkOutput(s.runtime, hc.Stdout)
				stderr := sink.NewSinkOutput(s.runtime, hc.Stderr)

				sinks := rt.NewTable()
				sinks.Set(rt.StringValue("in"), rt.UserDataValue(stdin.UserData))
				sinks.Set(rt.StringValue("input"), rt.UserDataValue(stdin.UserData))
				sinks.Set(rt.StringValue("out"), rt.UserDataValue(stdout.UserData))
				sinks.Set(rt.StringValue("err"), rt.UserDataValue(stderr.UserData))

				t := rt.NewThread(s.runtime)
				sig := make(chan os.Signal)
				exit := make(chan bool)

				luaexitcode := rt.IntValue(63)
				var err error
				go func() {
					defer func() {
						if r := recover(); r != nil {
							exit <- true
						}
					}()

					signal.Notify(sig, os.Interrupt)
					select {
						case <-sig:
							t.KillContext()
							return
					}

				}()

				go func() {
					luaexitcode, err = rt.Call1(t, rt.FunctionValue(cmd), rt.TableValue(luacmdArgs), rt.TableValue(sinks))
					exit <- true
				}()

				<-exit
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error in command:\n" + err.Error())
					return interp.NewExitStatus(1)
				}

				var exitcode uint8

				if code, ok := luaexitcode.TryInt(); ok {
					exitcode = uint8(code)
				} else if luaexitcode != rt.NilValue {
					// deregister commander
					delete(cmds, args[0])
					fmt.Fprintf(os.Stderr, "Commander did not return number for exit code. %s, you're fired.\n", args[0])
				}

				return interp.NewExitStatus(exitcode)
			}

			path, err := util.LookPath(args[0])
			if err == util.ErrNotExec {
				return util.ExecError{
					Typ: "not-executable",
					Cmd: args[0],
					Code: 126,
					Colon: true,
					Err: util.ErrNotExec,
				}
			} else if err != nil {
				return util.ExecError{
					Typ: "not-found",
					Cmd: args[0],
					Code: 127,
					Err: util.ErrNotFound,
				}
			}

			killTimeout := 2 * time.Second
			// from here is basically copy-paste of the default exec handler from
			// sh/interp but with our job handling

			env := hc.Env
			envList := os.Environ()
			env.Each(func(name string, vr expand.Variable) bool {
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

			//var j *job
			if bg {
				/*
				j = jobs.getLatest()
				j.setHandle(&cmd)
				err = j.start()
				*/
			} else {
				err = cmd.Start()
			}

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

				err = cmd.Wait()
			}

			exit := util.HandleExecErr(err)

			if bg {
				//j.exitCode = int(exit)
				//j.finish()
			}
			return interp.NewExitStatus(exit)
		})(s.runner)
		err = s.runner.Run(context.TODO(), stmt)
		if err != nil {
			return bg, strms.Stdout, strms.Stderr, err
		}
	}

	return bg, strms.Stdout, strms.Stderr, nil
}

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
