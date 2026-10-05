package dev

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/projectfs"
)

// Both generated-site and preview serving use the authoring file boundary.
// FileServer normalizes URL paths; the rooted operation also checks symlinks.
type projectHTTPFS struct {
	files *projectfs.FS
	base  string
}

func (f projectHTTPFS) Open(name string) (http.File, error) {
	return f.files.Open(filepath.Join(f.base, filepath.FromSlash(strings.TrimPrefix(name, "/"))))
}
