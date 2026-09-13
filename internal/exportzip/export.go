package exportzip

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/site"
)

type Options struct {
	Strict        bool
	IncludeDrafts bool
	Overwrite     bool
}

type Result struct {
	Archive     string               `json:"archive"`
	Filename    string               `json:"filename"`
	Pages       int                  `json:"pages"`
	Files       int                  `json:"files"`
	Bytes       int64                `json:"bytes"`
	DurationMS  int64                `json:"durationMs"`
	Diagnostics []content.Diagnostic `json:"diagnostics,omitempty"`
}

func DefaultFilename(projectDir string) string {
	name := strings.TrimSpace(filepath.Base(filepath.Clean(projectDir)))
	var safe strings.Builder
	lastDash := false
	for _, char := range strings.ToLower(name) {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9', char == '_':
			safe.WriteRune(char)
			lastDash = false
		case !lastDash:
			safe.WriteByte('-')
			lastDash = true
		}
	}
	name = strings.Trim(safe.String(), "-")
	if name == "" || name == "." {
		name = "site"
	}
	return name + ".zip"
}

func Create(projectDir, destination string, options Options) (Result, error) {
	started := time.Now()
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(destination) == "" {
		destination = filepath.Join(projectDir, DefaultFilename(projectDir))
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return Result{}, err
	}
	if info, statErr := os.Stat(destination); statErr == nil {
		if info.IsDir() {
			return Result{}, fmt.Errorf("export destination is a directory: %s", destination)
		}
		if !options.Overwrite {
			return Result{}, fmt.Errorf("archive already exists: %s (use --force to replace it)", destination)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return Result{}, statErr
	}
	parent := filepath.Dir(destination)
	if info, statErr := os.Stat(parent); statErr != nil || !info.IsDir() {
		return Result{}, fmt.Errorf("export directory does not exist: %s", parent)
	}

	temporaryRoot := filepath.Join(projectDir, ".mpress")
	if err := os.MkdirAll(temporaryRoot, 0o755); err != nil {
		return Result{}, err
	}
	workspace, err := os.MkdirTemp(temporaryRoot, "export-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(workspace)
	output := filepath.Join(workspace, "site")
	build, buildErr := site.Build(projectDir, site.BuildOptions{Strict: options.Strict, IncludeDrafts: options.IncludeDrafts, MinifyAssets: true, PurgeUnusedCSS: true, OutputDir: output})
	if buildErr != nil {
		return Result{Pages: build.Pages, Files: build.Files, Diagnostics: build.Diagnostics}, buildErr
	}

	temporary, err := os.CreateTemp(parent, ".mpress-export-*.zip")
	if err != nil {
		return Result{}, err
	}
	temporaryName := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryName)
		}
	}()
	if err := writeArchive(temporary, output); err != nil {
		_ = temporary.Close()
		return Result{}, err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return Result{}, err
	}
	if err := temporary.Close(); err != nil {
		return Result{}, err
	}
	if options.Overwrite {
		if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
	}
	if err := os.Rename(temporaryName, destination); err != nil {
		return Result{}, err
	}
	removeTemporary = false
	info, err := os.Stat(destination)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Archive: destination, Filename: filepath.Base(destination), Pages: build.Pages, Files: build.Files,
		Bytes: info.Size(), DurationMS: time.Since(started).Milliseconds(), Diagnostics: build.Diagnostics,
	}, nil
}

func writeArchive(destination io.Writer, source string) error {
	archive := zip.NewWriter(destination)
	walkErr := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cannot export non-regular file %s", path)
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		header.Method = zip.Deflate
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if walkErr != nil {
		_ = archive.Close()
		return walkErr
	}
	return archive.Close()
}
