package content

import (
	"math/rand"
	"strings"
	"testing"
)

func TestComponentLineScannerMatchesRegexp(t *testing.T) {
	rng := rand.New(rand.NewSource(44109))
	parts := []string{
		"plain", "", "   ", "\t", "@note", "@note ", "@note{x=1}",
		"@button[Go]", "@mention text", "@two-words", ":::note", "  :::",
		"text @note", "::", ":", "@9bad", "@bad_name", "@name\r",
	}
	for trial := 0; trial < 2000; trial++ {
		var source strings.Builder
		lines := 1 + rng.Intn(30)
		for line := 0; line < lines; line++ {
			source.WriteString(parts[rng.Intn(len(parts))])
			if line+1 < lines || rng.Intn(2) == 0 {
				source.WriteByte('\n')
			}
		}
		value := source.String()
		if got, want := hasComponentLine(value), componentLineRE.MatchString(value); got != want {
			t.Fatalf("component line scanner mismatch at trial %d for %q: got %t, want %t", trial, value, got, want)
		}
	}
}
