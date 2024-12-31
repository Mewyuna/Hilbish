package util

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"syscall"

	rt "github.com/arnodel/golua/runtime"
)

var ErrNotExec = errors.New("not executable")
var ErrNotFound = errors.New("not found")

type ExecError struct{
	Typ string
	Cmd string
	Code int
	Colon bool
	Err error
}

func (e ExecError) Error() string {
	return fmt.Sprintf("%s: %s", e.Cmd, e.Typ)
}

func (e ExecError) sprint() error {
	sep := " "
	if e.Colon {
		sep = ": "
	}

	return fmt.Errorf("hilbish: %s%s%s", e.Cmd, sep, e.Err.Error())
}

func IsExecError(err error) (ExecError, bool) {
	if exErr, ok := err.(ExecError); ok {
		return exErr, true
	}

	fields := strings.Split(err.Error(), ": ")
	knownTypes := []string{
		"not-found",
		"not-executable",
	}

	if len(fields) > 1 && Contains(knownTypes, fields[1]) {
		var colon bool
		var e error
		switch fields[1] {
			case "not-found":
				e = ErrNotFound
			case "not-executable":
				colon = true
				e = ErrNotExec
		}

		return ExecError{
			Cmd: fields[0],
			Typ: fields[1],
			Colon: colon,
			Err: e,
		}, true
	}

	return ExecError{}, false
}

// SetField sets a field in a table, adding docs for it.
// It is accessible via the __docProp metatable. It is a table of the names of the fields.
func SetField(rtm *rt.Runtime, module *rt.Table, field string, value rt.Value) {
	// TODO:    ^ rtm isnt needed, i should remove it
	module.Set(rt.StringValue(field), value)
}

// SetFieldProtected sets a field in a protected table. A protected table
// is one which has a metatable proxy to ensure no overrides happen to it.
// It sets the field in the table and sets the __docProp metatable on the
// user facing table.
func SetFieldProtected(module, realModule *rt.Table, field string, value rt.Value) {
	realModule.Set(rt.StringValue(field), value)
}

// DoString runs the code string in the Lua runtime.
func DoString(rtm *rt.Runtime, code string) (rt.Value, error) {
	chunk, err := rtm.CompileAndLoadLuaChunk("<string>", []byte(code), rt.TableValue(rtm.GlobalEnv()))
	var ret rt.Value
	if chunk != nil {
		ret, err = rt.Call1(rtm.MainThread(), rt.FunctionValue(chunk))
	}

	return ret, err
}

func MustDoString(rtm *rt.Runtime, code string) rt.Value {
	val, err := DoString(rtm, code)
	if err != nil {
		panic(err)
	}

	return val
}

// DoFile runs the contents of the file in the Lua runtime.
func DoFile(rtm *rt.Runtime, path string) error {
	f, err := os.Open(path)
	defer f.Close()

	if err != nil {
		return err
	}
	
	reader := bufio.NewReader(f)
	c, err := reader.ReadByte()
	if err != nil && err != io.EOF {
		return err
	}

	// unread so a char won't be missing
	err = reader.UnreadByte()
	if err != nil {
		return err
	}

	var buf []byte
	if c == byte('#') {
		// shebang - skip that line
		_, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		buf = []byte{'\n'}
	}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		
		buf = append(buf, line...)
	}

	clos, err := rtm.LoadFromSourceOrCode(path, buf, "bt", rt.TableValue(rtm.GlobalEnv()), false)
	if clos != nil {
		_, err = rt.Call1(rtm.MainThread(), rt.FunctionValue(clos))
	}

	return err
}

// HandleStrCallback handles function parameters for Go functions which take
// a string and a closure.
func HandleStrCallback(t *rt.Thread, c *rt.GoCont) (string, *rt.Closure, error) {
	if err := c.CheckNArgs(2); err != nil {
		return "", nil, err
	}
	name, err := c.StringArg(0)
	if err != nil {
		return "", nil, err
	}
	cb, err := c.ClosureArg(1)
	if err != nil {
		return "", nil, err
	}

	return name, cb, err
}

// ForEach loops through a Lua table.
func ForEach(tbl *rt.Table, cb func(key rt.Value, val rt.Value)) {
	nextVal := rt.NilValue
	for {
		key, val, _ := tbl.Next(nextVal)
		if key == rt.NilValue {
			break
		}
		nextVal = key

		cb(key, val)
	}
}

// ExpandHome expands ~ (tilde) in the path, changing it to the user home
// directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		curuser, _ := user.Current()
		homedir := curuser.HomeDir

		return strings.Replace(path, "~", homedir, 1)
	}

	return path
}

// AbbrevHome changes the user's home directory in the path string to ~ (tilde)
func AbbrevHome(path string) string {
	curuser, _ := user.Current()
	if strings.HasPrefix(path, curuser.HomeDir) {
		return "~" + strings.TrimPrefix(path, curuser.HomeDir)
	}

	return path
}

func LookPath(file string) (string, error) { // custom lookpath function so we know if a command is found *and* is executable
	var skip []string
	if runtime.GOOS == "windows" {
		skip = []string{"./", "../", "~/", "C:"}
	} else {
		skip = []string{"./", "/", "../", "~/"}
	}
	for _, s := range skip {
		if strings.HasPrefix(file, s) {
			return file, FindExecutable(file, false, false)
		}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		path := filepath.Join(dir, file)
		err := FindExecutable(path, true, false)
		if err == ErrNotExec {
			return "", err
		} else if err == nil {
			return path, nil
		}
	}

	return "", os.ErrNotExist
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if strings.ToLower(a) == strings.ToLower(e) {
			return true
		}
	}
	return false
}

func HandleExecErr(err error) (exit uint8) {
	ctx := context.TODO()

	switch x := err.(type) {
	case *exec.ExitError:
		// started, but errored - default to 1 if OS
		// doesn't have exit statuses
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				if ctx.Err() != nil {
					return
				}
				exit = uint8(128 + status.Signal())
				return
			}
			exit = uint8(status.ExitStatus())
			return
		}
		exit = 1
		return
	case *exec.Error:
		// did not start
		//fmt.Fprintf(hc.Stderr, "%v\n", err)
		exit = 127
	default: return
	}

	return
}
