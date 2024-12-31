package main

import (
	"fmt"
	"os"

	"hilbish/util"
	"hilbish/golibs/bait"
	"hilbish/golibs/commander"
	"hilbish/golibs/fs"
	"hilbish/golibs/snail"
	"hilbish/golibs/terminal"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/lib/debuglib"
)

var minimalconf = `hilbish.prompt '& '`

func luaInit() {
	l = rt.New(os.Stdout)
	l.PushContext(rt.RuntimeContextDef{
		MessageHandler: debuglib.Traceback,
	})
	lib.LoadAll(l)

	lib.LoadLibs(l, hilbishLoader)
	// yes this is stupid, i know
	util.DoString(l, "hilbish = require 'hilbish'")

	lib.LoadLibs(l, fs.Loader)
	lib.LoadLibs(l, terminal.Loader)
	lib.LoadLibs(l, snail.Loader)

	cmds = commander.New(l)
	lib.LoadLibs(l, cmds.Loader)

	hooks = bait.New(l)
	hooks.SetRecoverer(func(event string, handler *bait.Listener, err interface{}) {
		fmt.Println("Error in `error` hook handler:", err)
		hooks.Off(event, handler)
	})

	lib.LoadLibs(l, hooks.Loader)

	// Add Ctrl-C handler
	hooks.On("signal.sigint", func(...interface{}) {
		if !interactive {
			os.Exit(0)
		}
	})

	lr.rl.RawInputCallback = func(r []rune) {
		hooks.Emit("hilbish.rawInput", string(r))
	}

	// Add more paths that Lua can require from
	_, err := util.DoString(l, "package.path = package.path .. " + requirePaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not add Hilbish require paths! Libraries will be missing. This shouldn't happen.")
	}

	err1 := util.DoFile(l, "nature/init.lua")
	if err1 != nil {
		err2 := util.DoFile(l, preloadPath)
		if err2 != nil {
			fmt.Fprintln(os.Stderr, "Missing nature module, some functionality and builtins will be missing.")
			fmt.Fprintln(os.Stderr, "local error:", err1)
			fmt.Fprintln(os.Stderr, "global install error:", err2)
		}
	}
}

func runConfig(confpath string) {
	if !interactive {
		return
	}
	err := util.DoFile(l, confpath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err, "\nAn error has occured while loading your config! Falling back to minimal default config.")
		util.DoString(l, minimalconf)
	}
}
