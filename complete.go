package main

import (
	"errors"
	"path/filepath"
	"strings"
	"os"

	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
)

var charEscapeMap = []string{
	"\"", "\\\"",
	"'", "\\'",
	"`", "\\`",
	" ", "\\ ",
	"(", "\\(",
	")", "\\)",
	"[", "\\[",
	"]", "\\]",
	"$", "\\$",
	"&", "\\&",
	"*", "\\*",
	">", "\\>",
	"<", "\\<",
	"|", "\\|",
}
var charEscapeMapInvert = invert(charEscapeMap)
var escapeReplaer = strings.NewReplacer(charEscapeMap...)
var escapeInvertReplaer = strings.NewReplacer(charEscapeMapInvert...)

func invert(m []string) []string {
	newM := make([]string, len(charEscapeMap))
	for i := range m {
		if (i + 1) % 2 == 0 {
			newM[i] = m[i - 1]
			newM[i - 1] = m[i]
		}
	}

	return newM
}

func splitForFile(str string) []string {
	split := []string{}
	sb := &strings.Builder{}
	quoted := false

	for i, r := range str {
		if r == '"' {
			quoted = !quoted
			sb.WriteRune(r)
		} else if r == ' ' && str[i - 1] == '\\' {
			sb.WriteRune(r)
		} else if !quoted && r == ' ' {
			split = append(split, sb.String())
			sb.Reset()
		} else {
			sb.WriteRune(r)
		}
	}
	if strings.HasSuffix(str, " ") {
		split = append(split, "")
	}

	if sb.Len() > 0 {
		split = append(split, sb.String())
	}

	return split
}

func fileComplete(query, ctx string, fields []string) ([]string, string) {
	q := splitForFile(ctx)
	path := ""
	if len(q) != 0 {
		path = q[len(q) - 1]
	}

	return matchPath(path)
}

func binaryComplete(query, ctx string, fields []string) ([]string, string) {
	q := splitForFile(ctx)
	query = ""
	if len(q) != 0 {
		query = q[len(q) - 1]
	}

	var completions []string

	prefixes := []string{"./", "../", "/", "~/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(query, prefix) {
			fileCompletions, filePref := matchPath(query)
			if len(fileCompletions) != 0 {
				for _, f := range fileCompletions {
					fullPath, _ := filepath.Abs(util.ExpandHome(query + strings.TrimPrefix(f, filePref)))
					if err := util.FindExecutable(escapeInvertReplaer.Replace(fullPath), false, true); err != nil {
						continue
					}
					completions = append(completions, f)
				}
			}
			return completions, filePref
		}
	}

	// filter out executables, but in path
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		// search for an executable which matches our query string
		if matches, err := filepath.Glob(filepath.Join(dir, query + "*")); err == nil {
			// get basename from matches
			for _, match := range matches {
				// check if we have execute permissions for our match
				err := util.FindExecutable(match, true, false)
				if err != nil {
					continue
				}
				// get basename from match
				name := filepath.Base(match)
				// add basename to completions
				completions = append(completions, name)
			}
		}
	}

	// add lua registered commands to completions
	for cmdName := range cmds.Commands {
		if strings.HasPrefix(cmdName, query) {
			completions = append(completions, cmdName)
		}
	}

	completions = removeDupes(completions)

	return completions, query
}

func matchPath(query string) ([]string, string) {
	oldQuery := query
	query = strings.TrimPrefix(query, "\"")
	var entries []string
	var baseName string

	query = escapeInvertReplaer.Replace(query)
	path, _ := filepath.Abs(util.ExpandHome(filepath.Dir(query)))
	if string(query) == "" {
		// filepath base below would give us "."
		// which would cause a match of only dotfiles
		path, _ = filepath.Abs(".")
	} else if !strings.HasSuffix(query, string(os.PathSeparator)) {
		baseName = filepath.Base(query)
	}

	files, _ := os.ReadDir(path)
	for _, entry := range files {
		file, err := entry.Info()
		if err != nil {
			continue
		}

		if file.Mode() & os.ModeSymlink != 0 {
			path, err := filepath.EvalSymlinks(filepath.Join(path, file.Name()))
			if err == nil {
				file, err = os.Lstat(path)
			}
		}

		if strings.HasPrefix(file.Name(), baseName) {
			entry := file.Name()
			if file.IsDir() {
				entry = entry + string(os.PathSeparator)
			}
			if !strings.HasPrefix(oldQuery, "\"") {
				entry = escapeFilename(entry)
			}
			entries = append(entries, entry)
		}
	}
	if !strings.HasPrefix(oldQuery, "\"") {
		baseName = escapeFilename(baseName)
	}

	return entries, baseName
}

func escapeFilename(fname string) string {
	return escapeReplaer.Replace(fname)
}

// #interface completion
// tab completions
// The completions interface deals with tab completions.
func completionLoader(rtm *rt.Runtime) *rt.Table {
	exports := map[string]util.LuaExport{
		"bins": {hcmpBins, 3, false},
		"call": {hcmpCall, 4, false},
		"files": {hcmpFiles, 3, false},
		"handler": {hcmpHandler, 2, false},
	}

	mod := rt.NewTable()
	util.SetExports(rtm, mod, exports)
	
	return mod
}

// #interface completion
// bins(query, ctx, fields) -> entries (table), prefix (string)
// Return binaries/executables based on the provided parameters.
// This function is meant to be used as a helper in a command completion handler.
// #param query string
// #param ctx string
// #param fields table
/*
#example
-- an extremely simple completer for sudo.
hilbish.complete('command.sudo', function(query, ctx, fields)
	table.remove(fields, 1)
	if #fields[1] then
		-- return commands because sudo runs a command as root..!

		local entries, pfx = hilbish.completion.bins(query, ctx, fields)
		return {
			type = 'grid',
			items = entries
		}, pfx
	end

	-- ... else suggest files or anything else ..
end)
#example
*/
func hcmpBins(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	query, ctx, fds, err := getCompleteParams(t, c)
	if err != nil {
		return nil, err
	}

	completions, pfx := binaryComplete(query, ctx, fds)
	luaComps := rt.NewTable()

	for i, comp := range completions {
		luaComps.Set(rt.IntValue(int64(i + 1)), rt.StringValue(comp))
	}

	return c.PushingNext(t.Runtime, rt.TableValue(luaComps), rt.StringValue(pfx)), nil
}

// #interface completion
// call(name, query, ctx, fields) -> completionGroups (table), prefix (string)
// Calls a completer function. This is mainly used to call a command completer, which will have a `name`
// in the form of `command.name`, example: `command.git`.
// You can check the Completions doc or `doc completions` for info on the `completionGroups` return value.
// #param name string
// #param query string
// #param ctx string
// #param fields table
func hcmpCall(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(4); err != nil {
		return nil, err
	}
	completer, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	query, err := c.StringArg(1)
	if err != nil {
		return nil, err
	}
	ctx, err := c.StringArg(2)
	if err != nil {
		return nil, err
	}
	fields, err := c.TableArg(3)
	if err != nil {
		return nil, err
	}

	var completecb *rt.Closure
	var ok bool
	if completecb, ok = luaCompletions[completer]; !ok {
		return nil, errors.New("completer " + completer + " does not exist")
	}

	// we must keep the holy 80 cols
	cont := c.Next()
	err = rt.Call(l.MainThread(), rt.FunctionValue(completecb),
	[]rt.Value{rt.StringValue(query), rt.StringValue(ctx), rt.TableValue(fields)},
	cont)

	if err != nil {
		return nil, err
	}

	return cont, nil
}

// #interface completion
// files(query, ctx, fields) -> entries (table), prefix (string)
// Returns file matches based on the provided parameters.
// This function is meant to be used as a helper in a command completion handler.
// #param query string
// #param ctx string
// #param fields table
func hcmpFiles(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	query, ctx, fds, err := getCompleteParams(t, c)
	if err != nil {
		return nil, err
	}

	completions, pfx := fileComplete(query, ctx, fds)
	luaComps := rt.NewTable()

	for i, comp := range completions {
		luaComps.Set(rt.IntValue(int64(i + 1)), rt.StringValue(comp))
	}

	return c.PushingNext(t.Runtime, rt.TableValue(luaComps), rt.StringValue(pfx)), nil
}

// #interface completion
// handler(line, pos)
// This function contains the general completion handler for Hilbish. This function handles
// completion of everything, which includes calling other command handlers, binaries, and files.
// This function can be overriden to supply a custom handler. Note that alias resolution is required to be done in this function.
// #param line string The current Hilbish command line
// #param pos number Numerical position of the cursor
/*
#example
-- stripped down version of the default implementation
function hilbish.completion.handler(line, pos)
	local query = fields[#fields]

	if #fields == 1 then
		-- call bins handler here
	else
		-- call command completer or files completer here
	end
end
#example
*/
func hcmpHandler(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	return c.Next(), nil
}


func getCompleteParams(t *rt.Thread, c *rt.GoCont) (string, string, []string, error) {
	if err := c.CheckNArgs(3); err != nil {
		return "", "", []string{}, err
	}
	query, err := c.StringArg(0)
	if err != nil {
		return "", "", []string{}, err
	}
	ctx, err := c.StringArg(1)
	if err != nil {
		return "", "", []string{}, err
	}
	fields, err := c.TableArg(2)
	if err != nil {
		return "", "", []string{}, err
	}

	var fds []string
	util.ForEach(fields, func(k rt.Value, v rt.Value) {
		if v.Type() == rt.StringType {
			fds = append(fds, v.AsString())
		}
	})

	return query, ctx, fds, err
}
