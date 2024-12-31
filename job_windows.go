//go:build windows

package main

import (
	"errors"
	"syscall"
)

var bgProcAttr *syscall.SysProcAttr = &syscall.SysProcAttr{
	CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
}

func (j *job) foreground() error {
	return errors.New("not supported on windows")
}

func (j *job) background() error {
	return errors.New("not supported on windows")
}
