//go:build !windows && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd

package projectfs

import (
	"errors"
	"os"
)

func lockFile(*os.File, bool) error {
	return errors.New("snapshot process locking is unavailable on this platform")
}
