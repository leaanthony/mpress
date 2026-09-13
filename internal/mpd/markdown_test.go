package mpd

import (
	"strings"
	"testing"
)

func TestMarkdownOmitsEndForLeafComponents(t *testing.T) {
	document := Parse("leaf-components.mpd", []byte(`@image light="/light.svg" dark="/dark.svg" alt="Preview"
@qr url="https://example.com"
@input name="quantity" type="number" value=1
@computed expr="quantity * 2" deps="quantity"
`))
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == SeverityError {
			t.Fatalf("unexpected parse diagnostic %s: %s", diagnostic.Code, diagnostic.Message)
		}
	}
	output, err := Markdown(document)
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(output)
	for _, want := range []string{"@image{", "@qr{", "@input{", "@computed{"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("Markdown missing %q: %s", want, markdown)
		}
	}
	if strings.Contains(markdown, "@end") {
		t.Fatalf("leaf components emitted @end: %s", markdown)
	}
}
