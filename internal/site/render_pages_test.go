package site

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

func TestRenderPagesPropagatesPreparationError(t *testing.T) {
	want := errors.New("prepare page")
	_, _, err := renderPages(t.TempDir(), 1, false, false, func(int) (string, templateData, error) {
		return "", templateData{}, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("renderPages returned %v, want %v", err, want)
	}
}

func TestRenderPagesRejectsEscapedWriteTargets(t *testing.T) {
	for _, name := range []string{"../outside/index.html", "guide/../index.html", "/index.html", `guide\index.html`, "C:/index.html", "guide//index.html"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, root, "outside/index.html", "sentinel")
			_, _, err := renderPages(filepath.Join(root, "site"), 1, false, false, func(int) (string, templateData, error) {
				return name, templateData{Config: config.Default(), Page: &content.Page{Title: "Page", HTML: "<p>Page</p>"}}, nil
			})
			if err == nil {
				t.Error("unsafe output accepted")
			}
			data, err := os.ReadFile(filepath.Join(root, "outside/index.html"))
			if err != nil || string(data) != "sentinel" {
				t.Fatal("outside sentinel changed")
			}
			if _, err := os.Stat(filepath.Join(root, "site")); !os.IsNotExist(err) {
				t.Fatal("output created before rejection")
			}
		})
	}
}
