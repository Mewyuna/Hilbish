package main

import (
	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
)

// #interface editor
// interactions for Hilbish's line reader
// The hilbish.editor interface provides functions to
// directly interact with the line editor in use.
func editorLoader(rtm *rt.Runtime) *rt.Table {
	exports := map[string]util.LuaExport{
		/*
		"insert": {editorInsert, 1, false},
		"setVimRegister": {editorSetRegister, 1, false},
		"getVimRegister": {editorGetRegister, 2, false},
		"getLine": {editorGetLine, 0, false},
		"readChar": {editorReadChar, 0, false},
		*/
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, exports)

	return mod
}

// #interface editor
// insert(text)
// Inserts text into the Hilbish command line.
// #param text string
func editorInsert(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	text, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	lr.rl.Insert(text)

	return c.Next(), nil
}

// #interface editor
// setVimRegister(register, text)
// Sets the vim register at `register` to hold the passed text.
// #aram register string
// #param text string
func editorSetRegister(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	register, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	text, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}

	lr.rl.SetRegisterBuf(register, []rune(text))

	return c.Next(), nil
}

// #interface editor
// getVimRegister(register) -> string
// Returns the text that is at the register.
// #param register string
func editorGetRegister(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	register, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	buf := lr.rl.GetFromRegister(register)

	return c.PushingNext1(t.Runtime, rt.StringValue(string(buf))), nil
}

// #interface editor
// getLine() -> string
// Returns the current input line.
// #returns string
func editorGetLine(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	buf := lr.rl.GetLine()

	return c.PushingNext1(t.Runtime, rt.StringValue(string(buf))), nil
}

// #interface editor
// getChar() -> string
// Reads a keystroke from the user. This is in a format of something like Ctrl-L.
func editorReadChar(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	buf := lr.rl.ReadChar()

	return c.PushingNext1(t.Runtime, rt.StringValue(string(buf))), nil
}
