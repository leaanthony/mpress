//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package projectfs

import (
	"os"
	"syscall"
)

func lockFile(file *os.File, exclusive bool) error {
	operation := syscall.LOCK_SH
	if exclusive {
		operation = syscall.LOCK_EX
	}
	for {
		err := syscall.Flock(int(file.Fd()), operation)
		if err != syscall.EINTR {
			return err
		}
	}
}
