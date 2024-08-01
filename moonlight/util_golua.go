//go:build !midnight
package moonlight

import (
	"bufio"
	"io"
	"os"

	rt "github.com/arnodel/golua/runtime"
)

// DoString runs the code string in the Lua runtime.
func (mlr *Runtime) DoString(code string) (Value, error) {
	chunk, err := mlr.rt.CompileAndLoadLuaChunk("<string>", []byte(code), rt.TableValue(mlr.rt.GlobalEnv()))
	var ret rt.Value
	if chunk != nil {
		ret, err = rt.Call1(mlr.rt.MainThread(), rt.FunctionValue(chunk))
	}

	return ret, err
}

// DoFile runs the contents of the file in the Lua runtime.
func (mlr *Runtime) DoFile(path string) error {
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

	clos, err := mlr.rt.LoadFromSourceOrCode(path, buf, "bt", rt.TableValue(mlr.rt.GlobalEnv()), false)
	if clos != nil {
		_, err = rt.Call1(mlr.rt.MainThread(), rt.FunctionValue(clos))
	}

	return err
}
