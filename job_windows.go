// +build windows

package main

import (
	"errors"
)

func (j *job) foreground() error {
	return errors.New("not supported on windows")
}

func (j *job) background() error {
	return errors.New("not supported on windows")
}

func (j *job) suspend() error {
	return errors.New("not supported on windows")
}
