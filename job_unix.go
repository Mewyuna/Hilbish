// +build darwin linux

package main

import (
	"errors"
	"os"
	"syscall"
	
	"golang.org/x/sys/unix"
	rt "github.com/arnodel/golua/runtime"
)

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

func (j *job) suspend() error {
	proc := j.handle.Process
	if proc == nil {
		return nil
	}

	proc.Signal(syscall.SIGSTOP)
	hooks.Emit("job.suspend", rt.UserDataValue(j.ud))
	return nil
}
