package sink

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
)

var sinkMetaKey = rt.StringValue("hshsink")

// #type
// A sink is a structure that has input and/or output to/from
// a desination.
type Sink struct{
	Rw *bufio.ReadWriter
	file *os.File
	UserData *rt.UserData
	autoFlush bool
}

func Loader(rtm *rt.Runtime) *rt.Table {
	sinkMeta := rt.NewTable()

	sinkMethods := rt.NewTable()
	sinkFuncs := map[string]util.LuaExport{
		"flush": {luaSinkFlush, 1, false},
		"read": {luaSinkRead, 1, false},
		"readAll": {luaSinkReadAll, 1, false},
		"autoFlush": {luaSinkAutoFlush, 2, false},
		"write": {luaSinkWrite, 2, false},
		"writeln": {luaSinkWriteln, 2, false},
	}
	util.SetExports(rtm, sinkMethods, sinkFuncs)

	sinkIndex := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		s, _ := sinkArg(c, 0)

		arg := c.Arg(1)
		val := sinkMethods.Get(arg)

		if val != rt.NilValue {
			return c.PushingNext1(t.Runtime, val), nil
		}

		keyStr, _ := arg.TryString()

		switch keyStr {
			case "pipe":
				val = rt.BoolValue(false)
				if s.file != nil {
					fileInfo, _ := s.file.Stat();
					val = rt.BoolValue(fileInfo.Mode() & os.ModeCharDevice == 0)
				}
		}

		return c.PushingNext1(t.Runtime, val), nil
	}

	sinkMeta.Set(rt.StringValue("__index"), rt.FunctionValue(rt.NewGoFunction(sinkIndex, "__index", 2, false)))
	rtm.SetRegistry(sinkMetaKey, rt.TableValue(sinkMeta))

	exports := map[string]util.LuaExport{
		"new": {luaSinkNew, 0, false},
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, exports)

	return mod
}


func luaSinkNew(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	snk := NewSink(t.Runtime, new(bytes.Buffer))

	return c.PushingNext1(t.Runtime, rt.UserDataValue(snk.UserData)), nil
}

// #member
// readAll() -> string
// --- @returns string
// Reads all input from the sink.
func luaSinkReadAll(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	s, err := sinkArg(c, 0)
	if err != nil {
		return nil, err
	}

	lines := []string{}
	for {
		line, err := s.Rw.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		lines = append(lines, line)
	}

	return c.PushingNext1(t.Runtime, rt.StringValue(strings.Join(lines, ""))), nil
}

// #member
// read() -> string
// --- @returns string
// Reads a liine of input from the sink.
func luaSinkRead(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	s, err := sinkArg(c, 0)
	if err != nil {
		return nil, err
	}

	str, _ := s.Rw.ReadString('\n')

	return c.PushingNext1(t.Runtime, rt.StringValue(str)), nil
}

// #member
// write(str)
// Writes data to a sink.
func luaSinkWrite(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}

	s, err := sinkArg(c, 0)
	if err != nil {
		return nil, err
	}
	data, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}

	s.Rw.Write([]byte(data))
	if s.autoFlush {
		s.Rw.Flush()
	}

	return c.Next(), nil
}

// #member
// writeln(str)
// Writes data to a sink with a newline at the end.
func luaSinkWriteln(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}

	s, err := sinkArg(c, 0)
	if err != nil {
		return nil, err
	}
	data, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}

	s.Rw.Write([]byte(data + "\n"))
	if s.autoFlush {
		s.Rw.Flush()
	}

	return c.Next(), nil
}

// #member
// flush()
// Flush writes all buffered input to the sink.
func luaSinkFlush(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	s, err := sinkArg(c, 0)
	if err != nil {
		return nil, err
	}

	s.Rw.Flush()

	return c.Next(), nil
}

// #member
// autoFlush(auto)
// Sets/toggles the option of automatically flushing output.
// A call with no argument will toggle the value.
// --- @param auto boolean|nil
func luaSinkAutoFlush(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	s, err := sinkArg(c, 0)
	if err != nil {
		return nil, err
	}

	v := c.Arg(1)
	if v.Type() != rt.BoolType && v.Type() != rt.NilType {
		return nil, fmt.Errorf("#1 must be a boolean")
	}

	value := !s.autoFlush
	if v.Type() == rt.BoolType {
		value = v.AsBool()
	}

	s.autoFlush = value

	return c.Next(), nil
}

func NewSink(rtm *rt.Runtime, Rw io.ReadWriter) *Sink {
	s := &Sink{
		Rw: bufio.NewReadWriter(bufio.NewReader(Rw), bufio.NewWriter(Rw)),
	}
	s.UserData = sinkUserData(rtm, s)

	if f, ok := Rw.(*os.File); ok {
		s.file = f
	}

	return s
}

func NewSinkInput(rtm *rt.Runtime, r io.Reader) *Sink {
	s := &Sink{
		Rw: bufio.NewReadWriter(bufio.NewReader(r), nil),
	}
	s.UserData = sinkUserData(rtm, s)

	if f, ok := r.(*os.File); ok {
		s.file = f
	}

	return s
}

func NewSinkOutput(rtm *rt.Runtime, w io.Writer) *Sink {
	s := &Sink{
		Rw: bufio.NewReadWriter(nil, bufio.NewWriter(w)),
		autoFlush: true,
	}
	s.UserData = sinkUserData(rtm, s)

	return s
}

func sinkArg(c *rt.GoCont, arg int) (*Sink, error) {
	s, ok := valueToSink(c.Arg(arg))
	if !ok {
		return nil, fmt.Errorf("#%d must be a sink", arg + 1)
	}

	return s, nil
}

func valueToSink(val rt.Value) (*Sink, bool) {
	u, ok := val.TryUserData()
	if !ok {
		return nil, false
	}

	s, ok := u.Value().(*Sink)
	return s, ok
}

func sinkUserData(rtm *rt.Runtime, s *Sink) *rt.UserData {
	sinkMeta := rtm.Registry(sinkMetaKey)
	return rt.NewUserData(s, sinkMeta.AsTable())
}
