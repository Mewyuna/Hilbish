//go:build unix

package main

import (
	"errors"
	"os"
	"syscall"
	
	"golang.org/x/sys/unix"
)

var bgProcAttr *syscall.SysProcAttr = &syscall.SysProcAttr{
	Setpgid: true,
}

func (j *job) foreground() error {
	if jobs.foreground {
		return errors.New("(another) job already foregrounded")
	}

	pgid, _ := syscall.Getpgid(j.pid)
	// tcsetpgrp
	unix.IoctlSetPointerInt(0, unix.TIOCSPGRP, pgid)
	proc, _ := os.FindProcess(j.pid)
	proc.Wait()
	
	hshPgid, _ := syscall.Getpgid(os.Getpid())
	unix.IoctlSetPointerInt(0, unix.TIOCSPGRP, hshPgid)

	return nil
}

func (j *job) background() error {
	proc := j.handle.Process
	if proc == nil {
		return nil
	}

	proc.Signal(syscall.SIGCONT)
	return nil
}
