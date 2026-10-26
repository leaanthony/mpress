package deploy

import "github.com/leaanthony/mpress/internal/projectfs"

// outputRoot borrows an already pinned output boundary, or owns the explicitly
// selected standalone root. The no-op close keeps root lifetime with its owner.
func outputRoot(outputDir string, borrowed *projectfs.FS) (*projectfs.FS, func(), error) {
	if borrowed != nil {
		return borrowed, func() {}, nil
	}
	files, err := projectfs.Open(outputDir)
	if err != nil {
		return nil, nil, err
	}
	return files, func() { _ = files.Close() }, nil
}
