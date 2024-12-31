//go:build windows

package util

import (
	"path/filepath"
	"os"
)

func FindExecutable(path string, inPath, dirs bool) error {
	nameExt := filepath.Ext(path)
	pathExts := filepath.SplitList(os.Getenv("PATHEXT"))
	if inPath {
		if nameExt == "" {
			for _, ext := range pathExts {
				_, err := os.Stat(path + ext)
				if err == nil {
					return nil
				}
			}
		} else {
			_, err := os.Stat(path)
			if err == nil {
				if Contains(pathExts, nameExt) { return nil }
				return ErrNotExec
			}
		}
	} else {
		_, err := os.Stat(path)
		if err == nil {
			if Contains(pathExts, nameExt) { return nil }
			return ErrNotExec
		}
	}

	return os.ErrNotExist
}
