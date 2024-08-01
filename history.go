package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"hilbish/moonlight"
)

type luaHistory struct {}

func (h *luaHistory) Write(line string) (int, error) {
	histWrite := hshMod.Get(moonlight.StringValue("history")).AsTable().Get(moonlight.StringValue("add"))
	ln, err := l.Call1(histWrite, moonlight.StringValue(line))

	var num int64
	if ln.Type() == moonlight.IntType {
		num = ln.AsInt()
	}

	return int(num), err
}

func (h *luaHistory) GetLine(idx int) (string, error) {
	histGet := hshMod.Get(moonlight.StringValue("history")).AsTable().Get(moonlight.StringValue("get"))
	lcmd, err := l.Call1(histGet, moonlight.IntValue(int64(idx)))

	var cmd string
	if lcmd.Type() == moonlight.StringType {
		cmd = lcmd.AsString()
	}

	return cmd, err
}

func (h *luaHistory) Len() int {
	histSize := hshMod.Get(moonlight.StringValue("history")).AsTable().Get(moonlight.StringValue("size"))
	ln, _ := l.Call1(histSize)

	var num int64
	if ln.Type() == moonlight.IntType {
		num = ln.AsInt()
	}

	return int(num)
}

func (h *luaHistory) Dump() interface{} {
	// hilbish.history interface already has all function, this isnt used in readline
	return nil
}

type fileHistory struct {
	items []string
	f *os.File
}

func newFileHistory(path string) *fileHistory {
	dir := filepath.Dir(path)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			panic(err)
		}
	}

	lines := strings.Split(string(data), "\n")
	itms := make([]string, len(lines) - 1)
	for i, l := range lines {
		if i == len(lines) - 1 {
			continue
		}
		itms[i] = l
	}
	f, err := os.OpenFile(path, os.O_APPEND | os.O_WRONLY | os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}

	fh := &fileHistory{
		items: itms,
		f: f,
	}

	return fh
}

func (h *fileHistory) Write(line string) (int, error) {
	if line == "" {
		return len(h.items), nil
	}

	_, err := h.f.WriteString(line + "\n")
	if err != nil {
		return 0, err
	}
	h.f.Sync()

	h.items = append(h.items, line)
	return len(h.items), nil
}

func (h *fileHistory) GetLine(idx int) (string, error) {
	if len(h.items) == 0 {
		return "", nil
	}
	if idx == -1 { // this should be fixed readline side
		return "", nil
	}
	return h.items[idx], nil
}

func (h *fileHistory) Len() int {
	return len(h.items)
}

func (h *fileHistory) Dump() interface{} {
	return h.items
}

func (h *fileHistory) clear() {
	h.items = []string{}
	h.f.Truncate(0)
	h.f.Sync()
}
