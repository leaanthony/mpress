package projectfs

import (
	"io/fs"
	"runtime"
	"testing"
)

func TestAtomicWriteRetainsRequestedPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits do not apply on Windows")
	}
	files, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	for _, mode := range []fs.FileMode{0600, 0644, 0755} {
		if err := files.WriteAtomicMode("mode", []byte("owned data"), mode); err != nil {
			t.Fatal(err)
		}
		info, err := files.Stat("mode")
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("mode %o was not retained: %v %v", mode, info, err)
		}
	}
}
