//go:build !midnight
package main

import (
	"fmt"

	"hilbish/moonlight"
	rt "github.com/arnodel/golua/runtime"
)

func handleLua(input string) (string, uint8, error) {
	cmdString := aliases.Resolve(input)
	// First try to load input, essentially compiling to bytecode
	rtm := l.UnderlyingRuntime()
	chunk, err := rtm.CompileAndLoadLuaChunk("", []byte(cmdString), moonlight.TableValue(l.GlobalTable()))
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
			_, err = l.Call1(rt.FunctionValue(chunk))
		}
	}
	if err == nil {
		return cmdString, 0, nil
	}

	return cmdString, 125, err
}
