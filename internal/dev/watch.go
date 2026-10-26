package dev

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

const (
	// watchPollEvery bounds how long a saved change can go unnoticed.
	watchPollEvery = 500 * time.Millisecond
	// watchBufferBytes is reused for every file in a scan.
	watchBufferBytes = 64 << 10
)

func (s *Server) watch(ctx context.Context) {
	watchLoop(ctx, watchPollEvery, s.scanWatchState, func() { s.rebuild(true) })
}

// watchLoop compares complete input fingerprints. A change must survive one
// further scan before it is built, which coalesces a burst of writes. A scan
// error keeps the last good state; recovery from an initial scan error builds
// because no initial fingerprint was available.
func watchLoop(ctx context.Context, every time.Duration, scan func() (string, error), rebuild func()) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	var lastError string
	report := func(scanErr error) {
		if scanErr == nil {
			lastError = ""
			return
		}
		message := "watch: " + scanErr.Error()
		if message != lastError {
			fmt.Fprintln(os.Stderr, message)
			lastError = message
		}
	}

	baseline, err := scan()
	report(err)
	recovering := err != nil
	pending := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		next, scanErr := scan()
		if scanErr != nil {
			report(scanErr)
			continue
		}
		report(nil)
		if baseline == "" {
			baseline = next
			pending = ""
			if recovering {
				rebuild()
			}
			recovering = false
			continue
		}
		if next == baseline {
			pending = ""
			continue
		}
		if next == pending {
			baseline = next
			pending = ""
			rebuild()
			continue
		}
		pending = next
	}
}

// scanWatchState reloads the validated configuration and fingerprints the
// paths it selects. Reloading here keeps the watcher's watchlist current
// without mutating the server's shared configuration.
func (s *Server) scanWatchState() (string, error) {
	cfg, err := config.LoadWithReadFile(s.project, s.files.ReadFile)
	if err != nil {
		return "", fmt.Errorf("reload %s: %w", config.Filename, err)
	}
	fingerprint, err := s.watchFingerprint(cfg)
	if err != nil {
		return "", err
	}
	return fingerprint, nil
}

// watchInputs lists the configured files and directories whose content feeds a
// build. The navigation paths mirror site.Build, including a configured
// navFile that points outside the content directory and localized overrides.
// Output and state paths are deliberately absent: they are generated.
func watchInputs(cfg config.Config, project string) []string {
	content := cfg.ContentPath(project)
	inputs := []string{
		filepath.Join(project, config.Filename),
		content,
		cfg.StaticPath(project),
		filepath.Join(content, cfg.Build.NavFile),
	}
	for _, language := range cfg.Site.Languages {
		if language == cfg.Site.DefaultLanguage {
			continue
		}
		inputs = append(inputs, filepath.Join(content, language, cfg.Build.NavFile))
	}
	if cfg.Build.CustomCSS != "" {
		inputs = append(inputs, filepath.Join(project, cfg.Build.CustomCSS))
	}
	seen := make(map[string]bool, len(inputs))
	unique := make([]string, 0, len(inputs))
	for _, input := range inputs {
		cleaned := filepath.Clean(input)
		if seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		unique = append(unique, cleaned)
	}
	return unique
}

// watchFingerprint hashes the ordered logical names and full contents of the
// configured authoring inputs. A missing optional directory is itself part of
// the fingerprint, so creating it is observed.
func (s *Server) watchFingerprint(cfg config.Config) (string, error) {
	output := ""
	if rel, err := s.files.Relative(cfg.OutputPath(s.project)); err == nil {
		output = rel
	}
	scan := &watchScan{
		files:  s.files,
		digest: sha256.New(),
		output: output,
		buffer: make([]byte, watchBufferBytes),
	}
	_, _ = io.WriteString(scan.digest, "mpress-watch-v2\x00")
	for _, input := range watchInputs(cfg, s.project) {
		if err := scan.hashPath(input); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(scan.digest.Sum(nil)), nil
}

type watchScan struct {
	files  *projectfs.FS
	digest hash.Hash
	output string
	buffer []byte
}

func (scan *watchScan) hashPath(input string) error {
	rel, err := scan.files.Relative(input)
	if err != nil {
		return fmt.Errorf("watch %s: %w", input, err)
	}
	info, err := scan.files.Stat(rel)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(scan.digest, "missing\x00%s\x00", slashPath(rel))
			return nil
		}
		return fmt.Errorf("inspect %s: %w", rel, err)
	}
	if info.IsDir() {
		_, _ = fmt.Fprintf(scan.digest, "dir\x00%s\x00", slashPath(rel))
		return scan.hashDir(rel)
	}
	return scan.hashFile(rel)
}

func (scan *watchScan) hashDir(root string) error {
	return scan.files.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan %s: %w", name, err)
		}
		if entry.IsDir() {
			if name == root {
				return nil
			}
			if watchIgnoreDir(entry.Name(), name, scan.output) {
				return fs.SkipDir
			}
			_, _ = fmt.Fprintf(scan.digest, "dir\x00%s\x00", slashPath(name))
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return scan.hashAlias(name)
		}
		return scan.hashFile(name)
	})
}

// hashAlias includes a symbolic link's own identity and, through the confined
// filesystem, the content of a file it safely reaches inside the project.
func (scan *watchScan) hashAlias(name string) error {
	identity := slashPath(name)
	if link, err := scan.files.ReadLink(name); err == nil {
		_, _ = fmt.Fprintf(scan.digest, "alias\x00%s\x00%s\x00", identity, link)
	}
	info, err := scan.files.Stat(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(scan.digest, "link\x00%s\x00", identity)
			return nil
		}
		return fmt.Errorf("resolve %s: %w", name, err)
	}
	if info.IsDir() {
		// Directory aliases are not traversed, matching the build's own
		// non-following walk; their identity still participates.
		_, _ = fmt.Fprintf(scan.digest, "linkdir\x00%s\x00", identity)
		return nil
	}
	return scan.hashFile(name)
}

func (scan *watchScan) hashFile(name string) error {
	file, err := scan.files.Open(name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	defer func() { _ = file.Close() }()
	_, _ = fmt.Fprintf(scan.digest, "file\x00%s\x00", slashPath(name))
	if _, err := copyWatchFile(scan.digest, file, scan.buffer); err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	_, _ = io.WriteString(scan.digest, "\x00")
	return nil
}

// watchReader hides *os.File's WriteTo method so CopyBuffer reuses the scan
// buffer instead of letting the os package allocate one per file.
type watchReader struct{ io.Reader }

func copyWatchFile(digest hash.Hash, file io.Reader, buffer []byte) (int64, error) {
	return io.CopyBuffer(digest, watchReader{file}, buffer)
}

func watchIgnoreDir(name, path, output string) bool {
	switch name {
	case ".git", ".mpress":
		return true
	}
	if output == "" {
		return false
	}
	return path == output || strings.HasPrefix(path, output+string(filepath.Separator))
}

func slashPath(path string) string { return filepath.ToSlash(path) }
