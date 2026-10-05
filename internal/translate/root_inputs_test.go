package translate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

func TestBorrowedTranslationRootDoesNotEscapeToEngine(t *testing.T) {
	project := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	if err := os.MkdirAll(cfg.ContentPath(project), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.ContentPath(project), "index.mpd"), []byte("# Home\n\nBuild the compiler.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(project, cfg, &fakeProvider{})
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	files, err := projectfs.Open(project)
	if err != nil {
		t.Fatal(err)
	}
	_, runErr := engine.RunRoot(context.Background(), files, Options{Language: "fr", DryRun: true})
	borrowed, err := engine.BorrowRoot(files)
	if err != nil {
		t.Fatal(err)
	}
	_, auditErr := borrowed.Audit("fr", "index.mpd")
	document, err := ExtractMPD("index.mpd", []byte("# Home\n\nBuild the compiler.\n"))
	if err != nil {
		t.Fatal(err)
	}
	findings := []AuditFinding{{Severity: "warning", File: "index.mpd", Segment: document.Segments[1].ID, Message: "Improve grammar"}}
	_, refineErr := borrowed.RefineWithProvider(context.Background(), "fr", "index.mpd", findings, &fakeRefinementProvider{})
	_, checkErr := borrowed.Check(CheckOptions{})
	_, statErr := files.Stat("content/index.mpd")
	closeErr := files.Close()
	if runErr != nil || auditErr != nil || refineErr != nil || checkErr != nil || statErr != nil || closeErr != nil {
		t.Fatalf("borrowed-root ownership lost: run=%v audit=%v refine=%v check=%v stat=%v close=%v", runErr, auditErr, refineErr, checkErr, statErr, closeErr)
	}
	if _, err := engine.Run(context.Background(), Options{Language: "fr", DryRun: true}); err != nil {
		t.Fatalf("engine retained a closed borrowed root: %v", err)
	}
	if _, err := engine.Audit("fr", "index.mpd"); err != nil {
		t.Fatalf("audit retained a closed borrowed root: %v", err)
	}
	if _, err := engine.RefineWithProvider(context.Background(), "fr", "index.mpd", findings, &fakeRefinementProvider{}); err != nil {
		t.Fatalf("refinement retained a closed borrowed root: %v", err)
	}
	if _, err := engine.Check(CheckOptions{}); err != nil {
		t.Fatalf("check retained a closed borrowed root: %v", err)
	}
}

func TestRootedTranslationInputsRetainInternalAliases(t *testing.T) {
	project := t.TempDir()
	guide := filepath.Join(project, "guide.txt")
	if err := os.WriteFile(guide, []byte("writing guide"), 0600); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(project, "probe")
	if err := os.Symlink("guide.txt", probe); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	files, err := projectfs.Open(project)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	for _, target := range []string{"guide.txt", guide} {
		if err := os.Symlink(target, probe); err != nil {
			t.Fatal(err)
		}
		data, err := readOptionalProjectFileRoot(files, "probe", 13)
		if err != nil || data != "writing guide" {
			t.Fatalf("internal alias changed guide: %q %v", data, err)
		}
		if err := os.Remove(probe); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRootedTranslationInputRetainsBounds(t *testing.T) {
	project := t.TempDir()
	path := filepath.Join(project, "guide.txt")
	if err := os.WriteFile(path, []byte("writing guide"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := projectfs.Open(project)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	data, err := readProjectFileRoot(files, "guide.txt", 13)
	if err != nil || string(data) != "writing guide" {
		t.Fatalf("exact input boundary changed: %q %v", data, err)
	}
	if _, err := readProjectFileRoot(files, "guide.txt", 12); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized guide accepted: %v", err)
	}
	if _, err := readProjectFileRoot(files, "../guide.txt", 100); err == nil {
		t.Fatal("traversal accepted")
	}
	if _, err := readProjectFileRoot(files, project, 100); err == nil {
		t.Fatal("absolute input accepted")
	}
	if _, err := readProjectFileRoot(files, ".", 100); err == nil {
		t.Fatal("directory input accepted")
	}
}

// The old reader is retained only in this test unit, copied from cbb4c5c.
// Compare both readers on the same invalid sparse input; record rejection
// allocation cost, not ordinary load performance.
func BenchmarkOversizedTranslationInput(b *testing.B) {
	project := b.TempDir()
	file, err := os.Create(filepath.Join(project, "large.txt"))
	if err != nil {
		b.Fatal(err)
	}
	err = file.Truncate(16 << 20)
	closeErr := file.Close()
	if err != nil {
		b.Fatal(err)
	}
	if closeErr != nil {
		b.Fatal(closeErr)
	}
	files, err := projectfs.Open(project)
	if err != nil {
		b.Fatal(err)
	}
	defer files.Close()
	for _, name := range []string{"ambient", "rooted"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var err error
				if name == "ambient" {
					_, err = readProjectInputBefore(project, "large.txt", 128<<10)
				} else {
					_, err = readProjectFileRoot(files, "large.txt", 128<<10)
				}
				if err == nil {
					b.Fatal("oversized input accepted")
				}
			}
		})
	}
}

// Original cbb4c5c reader retained only as the rejection-cost baseline.
func readProjectInputBefore(root, name string, limit int64) ([]byte, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, errors.New("path must stay inside the project")
	}
	data, err := os.ReadFile(filepath.Join(root, clean))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return data, nil
}
