package site

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"
)

// parseCacheBuild names the mpress binary that wrote a cached page, so an
// upgrade re-renders every page even when parseCacheVersion was not bumped. A
// release is named by its module version and commit. A build from a working
// tree with uncommitted changes is also named by its executable's size and
// time, so rebuilding it after an edit starts afresh too.
var parseCacheBuild = sync.OnceValue(func() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return executableStamp()
	}
	parts := []string{info.Main.Version}
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			parts = append(parts, setting.Value)
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if modified || info.Main.Version == "" || info.Main.Version == "(devel)" && len(parts) == 1 {
		parts = append(parts, executableStamp())
	}
	return strings.Join(parts, " ")
})

func executableStamp() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	stat, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d %d", stat.Size(), stat.ModTime().UnixNano())
}
