// +build darwin linux

package main

import (
	"fmt"
	"syscall"
	"os"
	"os/signal"

	rt "github.com/arnodel/golua/runtime"
)

func handleSignals() {
	c := make(chan os.Signal)
	signal.Ignore(syscall.SIGTTOU, syscall.SIGTTIN)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGWINCH, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGQUIT, syscall.SIGTSTP)

	for s := range c {
		switch s {
			case os.Interrupt: hooks.Emit("signal.sigint")
			case syscall.SIGTERM: exit(0)
			case syscall.SIGWINCH: hooks.Emit("signal.resize")
			case syscall.SIGUSR1: hooks.Emit("signal.sigusr1")
			case syscall.SIGUSR2: hooks.Emit("signal.sigusr2")
			case syscall.SIGTSTP:
				suspendHandler := hshMod.Get(rt.StringValue("suspend"))
				_, err := rt.Call1(l.MainThread(), suspendHandler)
				if err != nil {
					fmt.Println(err)
				}
		}
	}
}
