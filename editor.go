package main

import (
	rt "github.com/arnodel/golua/runtime"
)

// #interface editor
// interactions for Hilbish's line reader
// The hilbish.editor interface provides functions to
// directly interact with the line editor in use.
func editorLoader(rtm *rt.Runtime) *rt.Table {
	return lr.rl.SetupLua(rtm)
}
