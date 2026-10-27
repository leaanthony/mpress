package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/leaanthony/mpress/internal/content"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A v23 parse-cache entry could hold a Page rendered before pricing and
// testimonial bodies were preserved. Rebuilding unchanged .mpd source must not
// keep serving that empty HTML from the old parse-cache epoch.
func TestBuildRefreshesCachedPreFixPricingAndTestimonialsHTML(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "site:\n  languages: [en]\nbuild:\n  contentDir: content\n")
	source := `# Plans

@pricing cols="2"
### Personal
$0
[Start](/guide/)
- ✓ First feature
---
### Team
$15/month
[More](/guide/)
recommended
- ✓ Shared work
@end

@testimonials autoplay="0"
"Fast and clear."
— Writer, Example team
---
"Reliable every day."
— Reader, Another team
@end
`
	writeFixture(t, root, "content/guide.mpd", source)

	cache := newParseCache(root)
	staleKey := sha256.Sum256([]byte("mpress-parse-v23\x00en\x00guide.mpd\x00" + source))
	staleKeyHex := hex.EncodeToString(staleKey[:])
	cache.saveWithQuickEdit(staleKeyHex, &content.Page{
		Language:   "en",
		URLPath:    "guide",
		OutputPath: "guide/index.html",
		Title:      "Plans",
		SourcePath: "guide.mpd",
		HTML:       `<h1>Plans</h1>`,
	}, nil, nil)
	if entry, ok := cache.loadEntry(staleKeyHex); !ok || entry.Page == nil || entry.Page.HTML != `<h1>Plans</h1>` {
		t.Fatalf("seeded v23 Glint cache entry missing or unreadable: %#v", entry.Page)
	}
	if _, err := os.Stat(filepath.Join(cache.dir, staleKeyHex+".glint")); err != nil {
		t.Fatalf("seeded v23 Glint cache file missing: %v", err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
			t.Fatalf("build %d: %v", attempt, err)
		}
		markup, err := os.ReadFile(filepath.Join(root, "site", "guide", "index.html"))
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(markup), `class="mpress-pricing-card`); got != 2 {
			t.Fatalf("build %d rendered %d pricing cards, want 2:\n%s", attempt, got, markup)
		}
		if got := strings.Count(string(markup), `<blockquote class="mpress-testimonial`); got != 2 {
			t.Fatalf("build %d rendered %d testimonial quotes, want 2:\n%s", attempt, got, markup)
		}
		for _, want := range []string{
			`<h3 class="mpress-pricing-name">Personal</h3>`,
			`<a href="/guide/" class="mpress-pricing-cta">Start</a>`,
			"First feature",
			`<div class="mpress-pricing-badge">Recommended</div>`,
			"Shared work",
			`“Fast and clear.”`,
			`<cite>Writer</cite>`,
			`<span class="mpress-testimonial-company">Example team</span>`,
			`“Reliable every day.”`,
			`<cite>Reader</cite>`,
			`<span class="mpress-testimonial-company">Another team</span>`,
		} {
			if !strings.Contains(string(markup), want) {
				t.Fatalf("build %d lost %q:\n%s", attempt, want, markup)
			}
		}
		current, err := os.ReadFile(filepath.Join(root, "content", "guide.mpd"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(current, []byte(source)) {
			t.Fatalf("build %d modified source bytes:\n%s", attempt, current)
		}
	}
}

func TestBuildReplacesCachedPreLocalizationHTML(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "site:\n  languages: [en, zh-cn]\nbuild:\n  contentDir: content\n")
	source := "# Guide\n\n从*[Go](https://go.dev/)*下载。\n\n```d2\n来源 -> 目标\n```\n"
	writeFixture(t, root, "content/zh-cn/guide.mpd", source)
	writeFixture(t, root, "content/index.md", "# Home\n")
	cache := newParseCache(root)
	legacy := sha256.Sum256([]byte("mpress-parse-v20\x00zh-cn\x00guide.mpd\x00" + source))
	stale, err := json.Marshal(parseCacheEntry{Page: &content.Page{
		Language: "zh-cn", URLPath: "guide", Title: "Guide", SourcePath: "guide.mpd",
		HTML: `<p>从**<a href="https://go.dev/">Go</a>**下载。</p>`,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache.dir, hex.EncodeToString(legacy[:])+".json"), stale, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	markup, err := os.ReadFile(filepath.Join(root, "site", "zh-cn", "guide", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markup), `从<strong><a href="https://go.dev/">Go</a></strong>下载。`) {
		t.Fatal("build reused pre-fix emphasis HTML")
	}
	if !strings.Contains(string(markup), `alt="来源, 目标"`) || strings.Contains(string(markup), `alt="Diagram:`) {
		t.Fatal("build reused the English diagram description")
	}
}

func TestNativeEmphasisSurvivesAdjacentTranslatedWords(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"*[go.dev/dl](https://go.dev/dl/)*에서 설치하세요.", `<strong><a href="https://go.dev/dl/">go.dev/dl</a></strong>에서`},
		{"_[下载](https://go.dev/dl/)_后安装。", `<em><a href="https://go.dev/dl/">下载</a></em>后`},
		{"これは*`Go`*です。", `これは<strong><code>Go</code></strong>です。`},
		{"Download *[Go](https://go.dev/dl/)* now.", `Download <strong><a href="https://go.dev/dl/">Go</a></strong> now.`},
	} {
		page, diagnostics, err := parseContentSource(content.NewRenderer(), "page.mpd", "ko", []byte(test.source+"\n"))
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range diagnostics {
			if d.Severity == "error" {
				t.Fatal(d)
			}
		}
		if !strings.Contains(page.HTML, test.want) {
			t.Fatalf("missing %q in %s", test.want, page.HTML)
		}
	}
}
