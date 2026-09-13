package highlight

import (
	"strings"
	"testing"
)

func TestShellHighlighting(t *testing.T) {
	tests := []struct {
		name     string
		language string
		line     string
		want     []string
	}{
		{
			name:     "command flags and inline comment",
			language: "bash",
			line:     "go install example.test/tool@latest --verbose # explain",
			want:     []string{`mpress-token-function">go</span>`, `mpress-token-keyword">--verbose</span>`, `mpress-token-comment"># explain</span>`},
		},
		{
			name:     "pipeline",
			language: "zsh",
			line:     `echo "$PATH" | grep go/bin`,
			want:     []string{`mpress-token-function">echo</span>`, `mpress-token-string">&#34;$PATH&#34;</span>`, `mpress-token-operator">|</span>`, `mpress-token-function">grep</span>`},
		},
		{
			name:     "powershell variable",
			language: "powershell",
			line:     `$env:PATH -split ';'`,
			want:     []string{`mpress-token-type">$env:PATH</span>`, `mpress-token-keyword">-split</span>`, `mpress-token-string">&#39;;&#39;</span>`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got strings.Builder
			WriteLine(&got, test.language, test.line)
			for _, want := range test.want {
				if !strings.Contains(got.String(), want) {
					t.Errorf("highlighted line missing %q:\n%s", want, got.String())
				}
			}
		})
	}
}
