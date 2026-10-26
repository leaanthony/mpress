package projectconvert

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

// configSnapshot records the configuration before document publication so a
// conflicting settings save can stop conversion before it replaces that leaf.
type configSnapshot struct {
	present bool
	regular bool
	alias   bool
	link    string
	data    []byte
	mode    os.FileMode
}

// captureConfigSnapshot also records initial absence: a file created by the
// author before the final check must not be mistaken for conversion output.
func captureConfigSnapshot(ops conversionOps, files *projectfs.FS) (configSnapshot, error) {
	info, err := ops.stat(files, config.Filename)
	if errors.Is(err, os.ErrNotExist) {
		return configSnapshot{}, nil
	}
	if err != nil {
		return configSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return configSnapshot{present: true}, nil
	}
	data, err := ops.readFile(files, config.Filename)
	if err != nil {
		return configSnapshot{}, fmt.Errorf("read configuration %s for conversion: %w", config.Filename, err)
	}
	snapshot := configSnapshot{present: true, regular: true, data: data, mode: info.Mode().Perm()}
	linkInfo, err := ops.lstat(files, config.Filename)
	if err != nil {
		return configSnapshot{}, fmt.Errorf("inspect configuration %s for conversion: %w", config.Filename, err)
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		snapshot.alias = true
		snapshot.link, err = ops.readLink(files, config.Filename)
		if err != nil {
			return configSnapshot{}, fmt.Errorf("read configuration alias %s for conversion: %w", config.Filename, err)
		}
	}
	return snapshot, nil
}

func configChangedDuringConversion() error {
	return fmt.Errorf("configuration %s changed during conversion; refusing to replace newer settings", config.Filename)
}

// unchanged reports whether the configuration at its original name still
// matches the planning snapshot. Presence, bytes, permissions and a final
// alias are all compared; an external alias already fails confinement while
// the snapshot is captured.
func (s configSnapshot) unchanged(ops conversionOps, files *projectfs.FS) error {
	info, err := ops.stat(files, config.Filename)
	if errors.Is(err, os.ErrNotExist) {
		if !s.present {
			return nil
		}
		return configChangedDuringConversion()
	}
	if err != nil {
		return err
	}
	if !s.present {
		return configChangedDuringConversion()
	}
	if !s.regular {
		if info.Mode().IsRegular() {
			return configChangedDuringConversion()
		}
		return nil
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != s.mode {
		return configChangedDuringConversion()
	}
	data, err := ops.readFile(files, config.Filename)
	if err != nil {
		return fmt.Errorf("read configuration %s during conversion: %w", config.Filename, err)
	}
	if !bytes.Equal(data, s.data) {
		return configChangedDuringConversion()
	}
	return s.sameAlias(ops, files, config.Filename)
}

// sameAlias compares the final leaf without resolving it, retaining the safe
// internal-alias semantics the conversion already supports.
func (s configSnapshot) sameAlias(ops conversionOps, files *projectfs.FS, name string) error {
	info, err := ops.lstat(files, name)
	if err != nil {
		return err
	}
	isAlias := info.Mode()&os.ModeSymlink != 0
	if isAlias != s.alias {
		return configChangedDuringConversion()
	}
	if !s.alias {
		return nil
	}
	link, err := ops.readLink(files, name)
	if err != nil {
		return err
	}
	if link != s.link {
		return configChangedDuringConversion()
	}
	return nil
}

// confirmCaptured compares the leaf moved into recovery with the planning
// snapshot. A relative alias is compared by its verbatim link text because its
// target may resolve differently after relocation; its bytes were compared
// before the move. A mismatch means the configuration was replaced between
// that comparison and the rename, so the captured newer leaf is left for
// rollback to restore instead of being overwritten.
func (s configSnapshot) confirmCaptured(ops conversionOps, files *projectfs.FS, backup string) error {
	info, err := ops.lstat(files, backup)
	if err != nil {
		return fmt.Errorf("inspect captured configuration %s: %w", backup, err)
	}
	if s.alias {
		if info.Mode()&os.ModeSymlink == 0 {
			return configChangedDuringConversion()
		}
		link, err := ops.readLink(files, backup)
		if err != nil {
			return fmt.Errorf("read captured configuration alias %s: %w", backup, err)
		}
		if link != s.link {
			return configChangedDuringConversion()
		}
		return nil
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return configChangedDuringConversion()
	}
	if info.Mode().Perm() != s.mode {
		return configChangedDuringConversion()
	}
	data, err := ops.readFile(files, backup)
	if err != nil {
		return fmt.Errorf("read captured configuration %s: %w", backup, err)
	}
	if !bytes.Equal(data, s.data) {
		return configChangedDuringConversion()
	}
	return nil
}
