package site

import (
	"github.com/leaanthony/mpress/internal/content"
	"strings"
	"testing"
)

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
