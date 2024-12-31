// filesystem interaction and functionality library
/*
The fs module provides filesystem functions to Hilbish. While Lua's standard
library has some I/O functions, they're missing a lot of the basics. The `fs`
library offers more functions and will work on any operating system Hilbish does.
#field pathSep The operating system's path separator.
*/
package fs

import (
	"fmt"
	"path/filepath"
	"strconv"
	"os"
	"strings"

	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
	"github.com/arnodel/golua/lib/packagelib"
	"github.com/arnodel/golua/lib/iolib"
)

var Loader = packagelib.Loader{
	Load: loaderFunc,
	Name: "fs",
}

func loaderFunc(rtm *rt.Runtime) (rt.Value, func()) {
	exports := map[string]util.LuaExport{
		"cd": util.LuaExport{fcd, 1, false},
		"mkdir": util.LuaExport{fmkdir, 2, false},
		"stat": util.LuaExport{fstat, 1, false},
		"readdir": util.LuaExport{freaddir, 1, false},
		"abs": util.LuaExport{fabs, 1, false},
		"basename": util.LuaExport{fbasename, 1, false},
		"dir": util.LuaExport{fdir, 1, false},
		"glob": util.LuaExport{fglob, 1, false},
		"join": util.LuaExport{fjoin, 0, true},
		"pipe": util.LuaExport{fpipe, 0, false},
	}
	mod := rt.NewTable()
	util.SetExports(rtm, mod, exports)
	mod.Set(rt.StringValue("pathSep"), rt.StringValue(string(os.PathSeparator)))
	mod.Set(rt.StringValue("pathListSep"), rt.StringValue(string(os.PathListSeparator)))

	return rt.TableValue(mod), nil
}

// abs(path) -> string
// Returns an absolute version of the `path`.
// This can be used to resolve short paths like `..` to `/home/user`.
// #param path string
// #returns string
func fabs(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	path = util.ExpandHome(path)

	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return c.PushingNext1(t.Runtime, rt.StringValue(abspath)), nil
}

// basename(path) -> string
// Returns the "basename," or the last part of the provided `path`. If path is empty,
// `.` will be returned.
// #param path string Path to get the base name of.
// #returns string
func fbasename(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	return c.PushingNext(t.Runtime, rt.StringValue(filepath.Base(path))), nil
}

// cd(dir)
// Changes Hilbish's directory to `dir`.
// #param dir string Path to change directory to.
func fcd(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	path = util.ExpandHome(strings.TrimSpace(path))

	err = os.Chdir(path)
	if err != nil {
		return nil, err
	}

	return c.Next(), err
}

// dir(path) -> string
// Returns the directory part of `path`. If a file path like
// `~/Documents/doc.txt` then this function will return `~/Documents`.
// #param path string Path to get the directory for.
// #returns string
func fdir(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	return c.PushingNext(t.Runtime, rt.StringValue(filepath.Dir(path))), nil
}

// glob(pattern) -> matches (table)
// Match all files based on the provided `pattern`.
// For the syntax' refer to Go's filepath.Match function: https://pkg.go.dev/path/filepath#Match
// #param pattern string Pattern to compare files with.
// #returns table A list of file names/paths that match.
/*
#example
--[[
	Within a folder that contains the following files:
	a.txt
	init.lua
	code.lua
	doc.pdf
]]--
local matches = fs.glob './*.lua'
print(matches)
-- -> {'init.lua', 'code.lua'}
#example
*/
func fglob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	pattern, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	luaMatches := rt.NewTable()

	for i, match := range matches {
		luaMatches.Set(rt.IntValue(int64(i + 1)), rt.StringValue(match))
	}
	
	return c.PushingNext(t.Runtime, rt.TableValue(luaMatches)), nil
}

// join(...path) -> string
// Takes any list of paths and joins them based on the operating system's path separator.
// #param path ...string Paths to join together
// #returns string The joined path.
/*
#example
-- This prints the directory for Hilbish's config!
print(fs.join(hilbish.userDir.config, 'hilbish'))
-- -> '/home/user/.config/hilbish' on Linux
#example
*/
func fjoin(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	strs := make([]string, len(c.Etc()))
	for i, v := range c.Etc() {
		if v.Type() != rt.StringType {
			// +2; go indexes of 0 and first arg from above
			return nil, fmt.Errorf("bad argument #%d to run (expected string, got %s)", i + 1, v.TypeName())
		}
		strs[i] = v.AsString()
	}

	res := filepath.Join(strs...)

	return c.PushingNext(t.Runtime, rt.StringValue(res)), nil
}

// mkdir(name, recursive)
// Creates a new directory with the provided `name`.
// With `recursive`, mkdir will create parent directories.
// #param name string Name of the directory
// #param recursive boolean Whether to create parent directories for the provided name
/*
#example
-- This will create the directory foo, then create the directory bar in the
-- foo directory. If recursive is false in this case, it will fail.
fs.mkdir('./foo/bar', true)
#example
*/
func fmkdir(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(2); err != nil {
		return nil, err
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	recursive, err := c.BoolArg(1)
	if err != nil {
		return nil, err
	}
	path = util.ExpandHome(strings.TrimSpace(path))

	if recursive {
		err = os.MkdirAll(path, 0744)
	} else {
		err = os.Mkdir(path, 0744)
	}
	if err != nil {
		return nil, err
	}

	return c.Next(), err
}

// fpipe() -> File, File
// Returns a pair of connected files, also known as a pipe.
// The type returned is a Lua file, same as returned from `io` functions.
// #returns File
// #returns File
func fpipe(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	rf, wf, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	rfLua := iolib.NewFile(rf, 0)
	wfLua := iolib.NewFile(wf, 0)

	return c.PushingNext(t.Runtime, rfLua.Value(t.Runtime), wfLua.Value(t.Runtime)), nil
}
// readdir(path) -> table[string]
// Returns a list of all files and directories in the provided path.
// #param dir string
// #returns table
func freaddir(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	dir, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	dir = util.ExpandHome(dir)
	names := rt.NewTable()

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for i, entry := range dirEntries {
		names.Set(rt.IntValue(int64(i + 1)), rt.StringValue(entry.Name()))
	}

	return c.PushingNext1(t.Runtime, rt.TableValue(names)), nil
}

// stat(path) -> {}
// Returns the information about a given `path`.
// The returned table contains the following values:
// name (string) - Name of the path
// size (number) - Size of the path in bytes
// mode (string) - Unix permission mode in an octal format string (with leading 0)
// isDir (boolean) - If the path is a directory
// #param path string
// #returns table
/*
#example
local inspect = require 'inspect'

local stat = fs.stat '~'
print(inspect(stat))
--[[
Would print the following:
{
  isDir = true,
  mode = "0755",
  name = "username",
  size = 12288
}
]]--
#example
*/
func fstat(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	path, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	path = util.ExpandHome(path)

	pathinfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	statTbl := rt.NewTable()
	statTbl.Set(rt.StringValue("name"), rt.StringValue(pathinfo.Name()))
	statTbl.Set(rt.StringValue("size"), rt.IntValue(pathinfo.Size()))
	statTbl.Set(rt.StringValue("mode"), rt.StringValue("0" + strconv.FormatInt(int64(pathinfo.Mode().Perm()), 8)))
	statTbl.Set(rt.StringValue("isDir"), rt.BoolValue(pathinfo.IsDir()))
	
	return c.PushingNext1(t.Runtime, rt.TableValue(statTbl)), nil
}

