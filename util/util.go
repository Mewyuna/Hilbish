package util

import (
	"strings"
	"os/user"

	"hilbish/moonlight"

	rt "github.com/arnodel/golua/runtime"
)

// SetField sets a field in a table, adding docs for it.
// It is accessible via the __docProp metatable. It is a table of the names of the fields.
func SetField(module *rt.Table, field string, value rt.Value) {
	module.Set(rt.StringValue(field), value)
}

// HandleStrCallback handles function parameters for Go functions which take
// a string and a closure.
func HandleStrCallback(mlr *moonlight.Runtime, c *moonlight.GoCont) (string, *moonlight.Closure, error) {
	if err := mlr.CheckNArgs(c, 2); err != nil {
		return "", nil, err
	}
	name, err := mlr.StringArg(c, 0)
	if err != nil {
		return "", nil, err
	}
	cb, err := mlr.ClosureArg(c, 1)
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
