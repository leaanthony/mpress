package projectfs

import (
	"os"

	"golang.org/x/sys/windows"
)

func lockFile(file *os.File, exclusive bool) error {
	var flags uint32
	if exclusive {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	return windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, ^uint32(0), ^uint32(0), &windows.Overlapped{})
}
