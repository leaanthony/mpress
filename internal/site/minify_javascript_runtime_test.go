package site

import (
	"os/exec"
	"strings"
	"testing"
)

func TestMinifiedScriptPreservesMethodAndDivisionResults(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required to compare JavaScript execution results")
	}
	programs := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "object method reads this and a parameter",
			source: `(() => {
  const counter = {
    base: 10,
    add(step) {
      return this.base + step;
    },
  };
  globalThis.result = counter.add(5);
})();`,
			want: "15",
		},
		{
			name: "class accessor pair keeps its value",
			source: `(() => {
  class Meter {
    constructor() {
      this.value = 2;
    }
    get doubled() {
      return this.value * 2;
    }
    set doubled(next) {
      this.value = next / 2;
    }
  }
  const meter = new Meter();
  meter.doubled = 12;
  globalThis.result = meter.doubled;
})();`,
			want: "12",
		},
		{
			name: "division produces fractional results",
			source: `(() => {
  const total = 84;
  const count = 4;
  globalThis.result = [total / count, 7 / 2];
})();`,
			want: "[21,3.5]",
		},
		{
			name: "division assignment updates the binding",
			source: `(() => {
  let remaining = 100;
  remaining /= 4;
  remaining /= 5;
  globalThis.result = remaining;
})();`,
			want: "5",
		},
		{
			name: "regex literal and division share one program",
			source: `(() => {
  const ratio = 12 / 3;
  const pattern = /a+b/g;
  const hits = "aaab ab".match(pattern);
  globalThis.result = [ratio, hits.length, hits[0]];
})();`,
			want: `[4,2,"aaab"]`,
		},
	}
	for _, program := range programs {
		t.Run(program.name, func(t *testing.T) {
			minified, err := minifyScript(program.source)
			if err != nil {
				t.Fatalf("minify script: %v", err)
			}
			if minified == strings.TrimSpace(program.source) {
				t.Fatal("script was not minified")
			}
			got, err := executeJavaScript(t, node, program.source)
			if err != nil {
				t.Fatalf("execute original: %v", err)
			}
			if got != program.want {
				t.Fatalf("original produced %q, want %q", got, program.want)
			}
			got, err = executeJavaScript(t, node, minified)
			if err != nil {
				t.Fatalf("execute minified %q: %v", minified, err)
			}
			if got != program.want {
				t.Fatalf("minified produced %q, want %q:\noriginal=%s\nminified=%s", got, program.want, program.source, minified)
			}
		})
	}
}
