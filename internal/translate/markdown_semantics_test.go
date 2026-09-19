package translate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestMarkdownEquivalentFormattingPreservesAuthoredTarget(t *testing.T) {
	for _, test := range []struct{ name, source, target string }{
		{"nested strong", "Use **not** and **never** here.", "ここでは****使用しません。また、使用不可です****。"},
		{"nested simplified Chinese", "No, I'm not joking: *No* *CGO* *dependency* 🤯!", "没错，我不是在开玩笑：*不再**依赖 CGO*** 🤯！"},
		{"nested traditional Chinese", "No, I'm not joking: *No* *CGO* *dependency* 🤯!", "不，我不是在開玩笑：*不再**依賴 CGO**了* 🤯！"},
		{"literal code font", "Yes (return false)", "Ja (`false` zurückgeben)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := ExtractMarkdown([]byte(test.source + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			target, err := ExtractMarkdown([]byte(test.target + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if err := Validate(source, target.Source); err != nil {
				t.Fatal(err)
			}
			values := map[string]string{}
			for i := range source.Segments {
				value, err := prepareExistingForFormat(source.Format, &source.Segments[i], target.Segments[i])
				if err != nil {
					t.Fatal(err)
				}
				values[source.Segments[i].ID] = value
			}
			output, err := Apply(source, values)
			if err != nil {
				t.Fatal(err)
			}
			if string(output) != string(target.Source) {
				t.Fatalf("changed authored target: %s", output)
			}
			if err := Validate(source, output); err != nil {
				t.Fatal(err)
			}
			if err := Validate(source, []byte(test.target+" `new command`\n")); err == nil {
				t.Fatal("reused target tokens weakened command validation")
			}
		})
	}
}

func TestMarkdownEquivalentFormattingRejectsProtectedChanges(t *testing.T) {
	for _, test := range []struct{ name, source, target string }{
		{"removed emphasis", "Use **not** and **never** here.", "Hier **niemals** verwenden."},
		{"literal stars", "Use **bold** and `***` here.", "Hier *****fett***** verwenden."},
		{"changed command", "Use **this** `run command`.", "Benutze **dies** `other command`."},
		{"changed URL", "Read **this** [guide](/guide/).", "Lies **diesen** [Text](/wrong/)."},
		{"changed boolean", "Yes (return false)", "Ja (`true` zurückgeben)"},
		{"not a word", "A falsehood", "Ein `false`"},
		{"duplicate boolean", "Yes (return false)", "Ja (`false` und `false`)"},
		{"code is not prose", "Use `false`", "Nutze `false` und `false`"},
		{"URL is not prose", "Read https://example.com/false", "Lies https://example.com/false und `false`"},
		{"new command font", "Use run command", "Benutze `run command`"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := ExtractMarkdown([]byte(test.source + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if err := Validate(source, []byte(test.target+"\n")); err == nil {
				t.Fatal("accepted changed protected content")
			}
		})
	}
}

type sourceRefinementProvider struct{ requests []TranslationRequest }

func (p *sourceRefinementProvider) Name() string  { return "source-refiner" }
func (p *sourceRefinementProvider) Model() string { return "test-model" }
func (p *sourceRefinementProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	p.requests = append(p.requests, request)
	result := map[string]string{}
	for _, segment := range request.Segments {
		result[segment.ID] = "Translated " + segment.Text
	}
	return result, nil
}

func TestMarkdownRefinementKeepsEnglishSourceAndUnselectedFormatting(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content", "fr"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "Yes (return false)\n\nUse **not** and **never** here.\n"
	target := "Ja (`false` zurückgeben)\n\nここでは****使用しません。また、使用不可です****。\n"
	targetPath := filepath.Join(root, "content", "fr", "page.md")
	for path, body := range map[string]string{filepath.Join(root, "content", "page.md"): source, targetPath: target} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	document, err := ExtractMarkdown([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	provider := &sourceRefinementProvider{}
	engine := NewEngine(root, cfg, nil)
	_, err = engine.RefineWithProvider(context.Background(), "fr", "page.md", []AuditFinding{{Severity: "warning", File: "page.md", Segment: document.Segments[0].ID, Message: "Improve this sentence"}}, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(provider.requests) != 1 || provider.requests[0].Segments[0].Text != document.Segments[0].Text {
		t.Fatalf("provider did not receive English source: %#v", provider.requests)
	}
	output, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "Translated Yes (return false)") || !strings.HasSuffix(string(output), strings.SplitN(target, "\n\n", 2)[1]) {
		t.Fatalf("wrong refinement or lost authored formatting: %s", output)
	}
}
