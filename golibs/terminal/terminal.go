// low level terminal library
// The terminal library is a simple and lower level library for certain terminal interactions.
package terminal

import (
	"os"

	"hilbish/moonlight"

	"golang.org/x/term"
)

var termState *term.State

func Loader(rtm *moonlight.Runtime) moonlight.Value {
	exports := map[string]moonlight.Export{
		"setRaw": {termsetRaw, 0, false},
		"restoreState": {termrestoreState, 0, false},
		"size": {termsize, 0, false},
		"saveState": {termsaveState, 0, false},
	}

	mod := moonlight.NewTable()
	rtm.SetExports(mod, exports)

	return moonlight.TableValue(mod)
}

// size()
// Gets the dimensions of the terminal. Returns a table with `width` and `height`
// NOTE: The size refers to the amount of columns and rows of text that can fit in the terminal.
func termsize(mlr *moonlight.Runtime, c *moonlight.GoCont) (moonlight.Cont, error) {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	dimensions := moonlight.NewTable()
	dimensions.SetField("width", moonlight.IntValue(int64(w)))
	dimensions.SetField("height", moonlight.IntValue(int64(h)))

	return mlr.PushNext1(c, moonlight.TableValue(dimensions)), nil
}

// saveState()
// Saves the current state of the terminal.
func termsaveState(mlr *moonlight.Runtime, c *moonlight.GoCont) (moonlight.Cont, error) {
	state, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	termState = state
	return c.Next(), nil
}

// restoreState()
// Restores the last saved state of the terminal
func termrestoreState(mlr *moonlight.Runtime, c *moonlight.GoCont) (moonlight.Cont, error) {
	err := term.Restore(int(os.Stdin.Fd()), termState)
	if err != nil {
		return nil, err
	}

	return c.Next(), nil
}

// setRaw()
// Puts the terminal into raw mode.
func termsetRaw(mlr *moonlight.Runtime, c *moonlight.GoCont) (moonlight.Cont, error) {
	_, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	return c.Next(), nil
}
