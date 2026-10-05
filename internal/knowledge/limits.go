package knowledge

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/leaanthony/mpress/internal/projectfs"
)

// Limits bound serialized input, not the larger parsed store or peak RSS.
// One budget is shared by the current bundle and every mounted snapshot.
type loadLimits struct {
	manifest, artifact, bundle, aggregate int64
	entries, versions                     int
}

func defaultLoadLimits() loadLimits {
	return loadLimits{manifest: 16 << 20, artifact: 256 << 20,
		bundle: 512 << 20, aggregate: 1 << 30, entries: 256, versions: 64}
}

type loadBudget struct {
	limits loadLimits
	used   int64
}

func newLoadBudget(limits loadLimits) *loadBudget { return &loadBudget{limits: limits} }

func (b *loadBudget) read(files *projectfs.FS, name string, maximum int64, bundle *int64) ([]byte, error) {
	maximum = min(maximum, b.limits.bundle-*bundle, b.limits.aggregate-b.used)
	data, err := readArtifactLimit(files, name, maximum)
	if err != nil {
		return nil, err
	}
	*bundle += int64(len(data))
	b.used += int64(len(data))
	return data, nil
}

func boundedVersionEntries(versions *projectfs.FS, limits loadLimits) ([]os.DirEntry, error) {
	directory, err := versions.Open(".")
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir(limits.entries + 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > limits.entries {
		return nil, fmt.Errorf("%w: versions exceeds %d entries", ErrResourceLimit, limits.entries)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	if count > limits.versions {
		return nil, fmt.Errorf("%w: versions exceeds %d directories", ErrResourceLimit, limits.versions)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}
