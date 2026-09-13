package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/leaanthony/mpress/internal/content"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReplacesCachedPreCJKEmphasisHTML(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "site:\n  languages: [en, zh-cn]\nbuild:\n  contentDir: content\n")
	source := "# Guide\n\n从*[Go](https://go.dev/)*下载。\n"
	writeFixture(t, root, "content/zh-cn/guide.mpd", source)
	writeFixture(t, root, "content/index.md", "# Home\n")
	cache := newParseCache(root)
	legacy := sha256.Sum256([]byte("mpress-parse-v19\x00zh-cn\x00guide.mpd\x00" + source))
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
