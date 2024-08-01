package main

import (
	"fmt"
	"io"
	"strings"

	"hilbish/moonlight"
	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
	"github.com/maxlandon/readline"
	"github.com/sahilm/fuzzy"
)

type lineReader struct {
	rl *readline.Instance
	fileHist *fileHistory
}
var hinter *rt.Closure
var highlighter *rt.Closure

func newLineReader(prompt string, noHist bool) *lineReader {
	rl := readline.NewInstance()
	lr := &lineReader{
		rl: rl,
	}

	regexSearcher := rl.Searcher
	rl.Searcher = func(needle string, haystack []string) []string {
		fz, _ := l.DoString("return hilbish.opts.fuzzy")
		fuzz, ok := fz.TryBool()
		if !fuzz || !ok {
			return regexSearcher(needle, haystack)
		}

		matches := fuzzy.Find(needle, haystack)
		suggs := make([]string, 0)

		for _, match := range matches {
			suggs = append(suggs, match.Str)
		}

		return suggs
	}

	// we don't mind hilbish.read rl instances having completion,
	// but it cant have shared history
	if !noHist {
		lr.fileHist = newFileHistory(defaultHistPath)
		rl.SetHistoryCtrlR("History", &luaHistory{})
		rl.HistoryAutoWrite = false
	}
	rl.ShowVimMode = false
	rl.ViModeCallback = func(mode readline.ViMode) {
		modeStr := ""
		switch mode {
			case readline.VimKeys: modeStr = "normal"
			case readline.VimInsert: modeStr = "insert"
			case readline.VimDelete: modeStr = "delete"
			case readline.VimReplaceOnce, readline.VimReplaceMany: modeStr = "replace"
		}
		setVimMode(modeStr)
	}
	rl.ViActionCallback = func(action readline.ViAction, args []string) {
		actionStr := ""
		switch action {
			case readline.VimActionPaste: actionStr = "paste"
			case readline.VimActionYank: actionStr = "yank"
		}
		hooks.Emit("hilbish.vimAction", actionStr, args)
	}
	rl.HintText = func(line []rune, pos int) []rune {
		hinter := hshMod.Get(moonlight.StringValue("hinter"))
		retVal, err := l.Call1(hinter, moonlight.StringValue(string(line)), moonlight.IntValue(int64(pos)))
		if err != nil {
			fmt.Println(err)
			return []rune{}
		}
		
		hintText := ""
		if luaStr, ok := retVal.TryString(); ok {
			hintText = luaStr
		}
		
		return []rune(hintText)
	}
	rl.SyntaxHighlighter = func(line []rune) string {
		highlighter := hshMod.Get(moonlight.StringValue("highlighter"))
		retVal, err := l.Call1(highlighter, moonlight.StringValue(string(line)))
		if err != nil {
			fmt.Println(err)
			return string(line)
		}
		
		highlighted := ""
		if luaStr, ok := retVal.TryString(); ok {
			highlighted = luaStr
		}
		
		return highlighted
	}
	setupTabCompleter(rl)

	return lr
}

func (lr *lineReader) Read() (string, error) {
	hooks.Emit("command.precmd", nil)
	s, err := lr.rl.Readline()
	// this is so dumb
	if err == readline.EOF {
		fmt.Println("")
		return "", io.EOF
	}

	return s, err // might get another error
}

func (lr *lineReader) SetPrompt(p string) {
	halfPrompt := strings.Split(p, "\n")
	if len(halfPrompt) > 1 {
		lr.rl.Multiline = true
		lr.rl.SetPrompt(strings.Join(halfPrompt[:len(halfPrompt) - 1], "\n"))
		lr.rl.MultilinePrompt = halfPrompt[len(halfPrompt) - 1:][0]
	} else {
		lr.rl.Multiline = false
		lr.rl.MultilinePrompt = ""
		lr.rl.SetPrompt(p)
	}
	if initialized && !running {
		lr.rl.RefreshPromptInPlace("")
	}
}

func (lr *lineReader) SetRightPrompt(p string) {
	lr.rl.SetRightPrompt(p)
	if initialized && !running {
		lr.rl.RefreshPromptInPlace("")
	}
}

func (lr *lineReader) AddHistory(cmd string) {
	lr.fileHist.Write(cmd)
}

func (lr *lineReader) ClearInput() {
	return
}

func (lr *lineReader) Resize() {
	return
}

// #interface history
// command history
// The history interface deals with command history. 
// This includes the ability to override functions to change the main
// method of saving history.
func (lr *lineReader) Loader(rtm *rt.Runtime) *rt.Table {
	lrLua := map[string]util.LuaExport{
		/*
		"add": {lr.luaAddHistory, 1, false},
		"all": {lr.luaAllHistory, 0, false},
		"clear": {lr.luaClearHistory, 0, false},
		"get": {lr.luaGetHistory, 1, false},
		"size": {lr.luaSize, 0, false},
		*/
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, lrLua)

	return mod
}

// #interface history
// add(cmd)
// Adds a command to the history.
// #param cmd string
func (lr *lineReader) luaAddHistory(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	cmd, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	lr.AddHistory(cmd)

	return c.Next(), nil
}

// #interface history
// size() -> number
// Returns the amount of commands in the history.
// #eturns number
func (lr *lineReader) luaSize(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(lr.fileHist.Len()))), nil
}

// #interface history
// get(index)
// Retrieves a command from the history based on the `index`.
// #param index number
func (lr *lineReader) luaGetHistory(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	idx, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}

	cmd, _ := lr.fileHist.GetLine(int(idx))

	return c.PushingNext1(t.Runtime, rt.StringValue(cmd)), nil
}

// #interface history
// all() -> table
// Retrieves all history as a table.
// #returns table
func (lr *lineReader) luaAllHistory(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	tbl := rt.NewTable()
	size := lr.fileHist.Len()

	for i := 1; i < size; i++ {
		cmd, _ := lr.fileHist.GetLine(i)
		tbl.Set(rt.IntValue(int64(i)), rt.StringValue(cmd))
	}

	return c.PushingNext1(t.Runtime, rt.TableValue(tbl)), nil
}

// #interface history
// clear()
// Deletes all commands from the history.
func (lr *lineReader) luaClearHistory(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	lr.fileHist.clear()
	return c.Next(), nil
}
