package quickedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/translate"
)

func TestExtractReusesTranslationSegments(t *testing.T) {
	source := []byte("---\ntitle: Guide\n---\n\n## Install\n\nInstall M-Press quickly.\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	if document.Revision != Revision(source) || len(document.Segments) != 2 {
		t.Fatalf("unexpected quick-edit document: %#v", document)
	}
	if document.Segments[0].ID != "b0001-heading-01" || document.Segments[1].Original != "Install M-Press quickly." {
		t.Fatalf("translation segment identity was not preserved: %#v", document.Segments)
	}
}

func TestExtractEditableMatchesFullTranslationSegmentIdentity(t *testing.T) {
	source := []byte("---\ntitle: Guide\ndescription: A {product} guide.\n---\n\n# Install **M-Press**\n\n1. Read the [guide](/guide/).\n2. Run `mpress build`.\n\n:::note\nKeep {product} current.\n:::\n\n```go\nfmt.Println(\"not editable\")\n```\n")
	full, err := translate.Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	editable, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	var visible []translate.Segment
	for _, segment := range full.Segments {
		if segment.Kind != "frontmatter" && strings.TrimSpace(segment.Original) != "" {
			visible = append(visible, segment)
		}
	}
	if len(editable.Segments) != len(visible) {
		t.Fatalf("editable segments = %d, full visible segments = %d", len(editable.Segments), len(visible))
	}
	for i, want := range visible {
		got := editable.Segments[i]
		if got.ID != want.ID || got.Kind != want.Kind || got.Original != want.Original || got.Start != want.Start || got.End != want.End || got.SourceHash != want.SourceHash {
			t.Fatalf("segment %d differs: got %#v, want %#v", i, got, want)
		}
	}
}

func TestDraftRoundTripAndApply(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "content", "guide.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	source := []byte("---\ntitle: Guide\n---\n\nOld sentence.\n")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "Old sentence.")
	payload, err := Marshal(Draft{Version: Version, Source: "content/guide.md", Revision: Revision(source), Changes: []Change{
		{ID: "b0001-text-01", Start: start, End: start + len("Old sentence."), Original: "Old sentence.", Markdown: "**Better** _sentence_ with [details](./details/)."},
		{ID: "insert:b0001-text", Start: len(source), End: len(source), Markdown: "\nA new paragraph.\n"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	draftPath := filepath.Join(root, "change.mpress-draft")
	if err := os.WriteFile(draftPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	count, err := ApplyFile(root, "content/guide.md", draftPath)
	if err != nil || count != 2 {
		t.Fatalf("apply: count=%d err=%v", count, err)
	}
	updated, _ := os.ReadFile(path)
	if got := string(updated); !strings.Contains(got, "**Better** _sentence_ with [details](./details/).") || !strings.Contains(got, "A new paragraph.") {
		t.Fatalf("draft was not applied: %s", got)
	}
}

func TestApplyRejectsStaleAndEscapingDrafts(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "guide.md")
	if err := os.WriteFile(path, []byte("Current\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, _ := Marshal(Draft{Version: Version, Source: "guide.md", Revision: Revision([]byte("Old\n")), Changes: []Change{{Start: 0, End: 3, Original: "Old", Markdown: "New"}}})
	stalePath := filepath.Join(root, "stale.mpress-draft")
	if err := os.WriteFile(stalePath, stale, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyFile(root, "guide.md", stalePath); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("stale draft was accepted: %v", err)
	}
	escape, _ := Marshal(Draft{Version: Version, Source: "../guide.md", Revision: Revision([]byte("Current\n")), Changes: []Change{{Start: 0, End: 7, Original: "Current", Markdown: "Changed"}}})
	escapePath := filepath.Join(root, "escape.mpress-draft")
	if err := os.WriteFile(escapePath, escape, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyFile(root, "../guide.md", escapePath); err == nil {
		t.Fatal("escaping draft was accepted")
	}
}

func TestResolveDraftFileFindsConfiguredDownloadsDirectory(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".config")
	downloads := filepath.Join(home, "My Downloads")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_DOWNLOAD_DIR", "")
	if err := os.WriteFile(filepath.Join(config, "user-dirs.dirs"), []byte("XDG_DOWNLOAD_DIR=\"$HOME/My Downloads\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	name := "mpress-draft-test.mpress-draft"
	want := filepath.Join(downloads, name)
	if err := os.WriteFile(want, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveDraftFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ResolveDraftFile() = %q, want %q", got, want)
	}
}
