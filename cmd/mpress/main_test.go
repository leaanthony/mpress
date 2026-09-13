package main

import (
	"archive/zip"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/site"
)

func TestKnowledgeHTTPBindingAndTokenSafety(t *testing.T) {
	got := interspersedKnowledgeArgs([]string{"./docs", "--transport", "http", "--no-build", "--port=3101"})
	want := []string{"--transport", "http", "--no-build", "--port=3101", "./docs"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("interspersed knowledge args=%q, want %q", got, want)
	}
	got = interspersedKnowledgeEvaluationArgs([]string{"./docs", "--suite", "suite.json", "--json", "--limit=10"})
	want = []string{"--suite", "suite.json", "--json", "--limit=10", "./docs"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("interspersed evaluation args=%q, want %q", got, want)
	}
	for host, want := range map[string]bool{"127.0.0.1": true, "::1": true, "localhost": true, "0.0.0.0": false, "docs.example.test": false} {
		if got := loopbackHost(host); got != want {
			t.Errorf("loopbackHost(%q)=%v, want %v", host, got, want)
		}
	}
	next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusNoContent) })
	handler := requireKnowledgeToken(next, "secret")
	for name, test := range map[string]struct {
		value string
		code  int
	}{
		"missing": {"", http.StatusUnauthorized},
		"wrong":   {"Bearer incorrect", http.StatusUnauthorized},
		"valid":   {"Bearer secret", http.StatusNoContent},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if test.value != "" {
				request.Header.Set("Authorization", test.value)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.code {
				t.Fatalf("status=%d, want %d", response.Code, test.code)
			}
		})
	}
}

func TestPrepareDevProjectClonesAndCreatesAWorkBranch(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if err := initProject([]string{source}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main", source},
		{"-C", source, "config", "user.email", "mpress@example.test"},
		{"-C", source, "config", "user.name", "M-Press Test"},
		{"-C", source, "add", "."},
		{"-C", source, "commit", "-m", "Initial site"},
	} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	destination := filepath.Join(t.TempDir(), "checkout")
	root, err := prepareDevProject(source, destination, "docs/add-french", nil)
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, _ := filepath.Abs(destination)
	if root != wantRoot {
		t.Fatalf("prepared root %q, want %q", root, wantRoot)
	}
	output, err := exec.Command("git", "-C", root, "branch", "--show-current").Output()
	if err != nil {
		t.Fatal(err)
	}
	if branch := strings.TrimSpace(string(output)); branch != "docs/add-french" {
		t.Fatalf("prepared branch %q", branch)
	}
	if _, err := config.Load(root); err != nil {
		t.Fatalf("prepared checkout is not an M-Press project: %v", err)
	}
}

func TestPrepareDevProjectAcceptsAnExistingProjectDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "site")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "content")
	found, err := prepareDevProject("", "", "docs/mpress", []string{nested})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(root)
	if found != want {
		t.Fatalf("found project %q, want %q", found, want)
	}
}

func TestContributionStartCommitRemainsTheSourceBranchBase(t *testing.T) {
	root := filepath.Join(t.TempDir(), "site")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main", root},
		{"-C", root, "config", "user.email", "mpress@example.test"},
		{"-C", root, "config", "user.name", "M-Press Test"},
		{"-C", root, "add", "."},
		{"-C", root, "commit", "-m", "Initial site"},
	} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	baseOutput, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSpace(string(baseOutput))
	if output, err := exec.Command("git", "-C", root, "checkout", "-b", "contribute/test").CombinedOutput(); err != nil {
		t.Fatalf("create contribution branch: %v\n%s", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, "content", "index.md"), []byte("# Changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-C", root, "add", "content/index.md"}, {"-C", root, "commit", "-m", "Change docs"}} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	if got := contributionStartCommit(root, "main"); got != base {
		t.Fatalf("contribution start commit = %q, want source base %q", got, base)
	}
}

func TestContributionOptionsWorkBeforeOrAfterThePageURL(t *testing.T) {
	for name, args := range map[string][]string{
		"options first": {"--branch", "next", "--checkout", "/tmp/docs", "--port=4123", "--draft-file", "change.mpress-draft", "--goal", "translate", "--no-open", "https://docs.example.test/guide/"},
		"URL first":     {"https://docs.example.test/guide/", "--no-open", "--port", "4123", "--checkout=/tmp/docs", "--branch=next", "--draft-file=change.mpress-draft", "--goal=translate"},
	} {
		t.Run(name, func(t *testing.T) {
			options, err := parseContributionArgs(args)
			if err != nil {
				t.Fatal(err)
			}
			if options.SiteURL != "https://docs.example.test/guide/" || options.Branch != "next" || options.Checkout != "/tmp/docs" || options.Port != 4123 || options.DraftFile != "change.mpress-draft" || options.Goal != "translate" || !options.NoOpen || options.Host != "127.0.0.1" {
				t.Fatalf("unexpected options: %#v", options)
			}
		})
	}
}

func TestInitCreatesGuidedTutorialProjectWithoutOverwriting(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my-docs")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"mpress.yaml", "content/index.md", "content/getting-started.md", "content/components.md", "content/_nav.yaml", "static/images/component-light.svg", "static/images/component-dark.svg", ".gitignore", ".mpress/onboarding"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatalf("missing initial project file %s: %v", name, err)
		}
	}
	guide, _ := os.ReadFile(filepath.Join(root, "content", "getting-started.md"))
	for _, want := range []string{"Edit this page", "@terminal", "Run checks"} {
		if !strings.Contains(string(guide), want) {
			t.Errorf("tutorial does not include %q", want)
		}
	}
	home, _ := os.ReadFile(filepath.Join(root, "content", "index.md"))
	for _, want := range []string{"@button[Start project setup]", "#your-first-three-steps", "icon=play"} {
		if !strings.Contains(string(home), want) {
			t.Errorf("starter home page does not include %q", want)
		}
	}
	componentPage, _ := os.ReadFile(filepath.Join(root, "content", "components.md"))
	for _, want := range []string{
		"@tabs", "@terminal", "@note", "@details", "@badge", "@api{", "@steps", "@cards", "@diff", "@filetree", "@linkcard",
		"@calendar", "@changelog", "@matrix", "@tutorial", "@status", "@pricing", "@release", "@testimonials", "@api-playground",
		"@qr", "@audience", "@explained", "@input", "@computed", "@if", "@variant", "@container", "@section", "@columns",
		"@column", "@actions", "@headline", "@docs-preview", "@preview-tabs", "@callout", "@timeline", "@capabilities", "@resources", "@image", "@button",
	} {
		if !strings.Contains(string(componentPage), want) {
			t.Errorf("starter component catalogue does not include %q", want)
		}
	}
	if strings.Contains(string(componentPage), ":::") {
		t.Fatalf("starter component catalogue contains the deprecated three-colon grammar")
	}
	if result, err := site.Build(root, site.BuildOptions{Strict: true}); err != nil {
		t.Fatalf("starter project does not build strictly: %v; diagnostics: %#v", err, result.Diagnostics)
	}
	configuration, _ := os.ReadFile(filepath.Join(root, "mpress.yaml"))
	for _, want := range []string{
		"languageLabels:", "logoLight:", "socialImage:", "headerLinks:", "social:", "reddit:", "rss:", "sponsor:",
		"contribution:", "customCSS:", "hoverColorLight:", "hoverColorDark:", "contentWidth:", "wideContentWidth:",
		"sidebarWidth:", "tocWidth:", "contentTocGap:", "alignment:", "toc:", "accessibility:", "shortcut: Mod+K", "shortcut: Mod+A", "versioning:", "translation:", "deploy:",
	} {
		if !strings.Contains(string(configuration), want) {
			t.Errorf("starter configuration does not include %q", want)
		}
	}
	if err := initProject([]string{root}); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected overwrite protection, got %v", err)
	}
}

func TestInitNextStepsAreRunnableAndExplainTheProjectGuide(t *testing.T) {
	steps := initNextSteps("/tmp/M-Press user's site")
	for _, want := range []string{"Next:", "cd '/tmp/M-Press user'\"'\"'s site'", "mpress dev", "opens the project guide"} {
		if !strings.Contains(steps, want) {
			t.Fatalf("next steps %q do not contain %q", steps, want)
		}
	}
}

func TestConfigureDeployPersistsACloudflareTargetWithoutASecret(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deployment-docs")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	if err := configureDeploy([]string{"cloudflare", "--account", "account-123", "--project", "product-docs", "--production-branch", "trunk"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	target := cfg.Deploy.Targets["cloudflare"]
	if cfg.Deploy.Default != "cloudflare" || target.AccountID != "account-123" || target.Project != "product-docs" || target.ProductionBranch != "trunk" {
		t.Fatalf("unexpected deployment configuration: %#v", cfg.Deploy)
	}
	data, err := os.ReadFile(filepath.Join(root, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(data)), "token") {
		t.Fatalf("deployment configuration must not contain a token: %s", data)
	}
}

func TestExportCommandCreatesAProductionZIP(t *testing.T) {
	root := filepath.Join(t.TempDir(), "portable-docs")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	archivePath := filepath.Join(t.TempDir(), "portable.zip")
	if err := exportSite([]string{archivePath, "--strict", "--force"}); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	foundIndex := false
	for _, item := range archive.File {
		if item.Name == "index.html" {
			foundIndex = true
		}
		if strings.Contains(item.Name, "__mpress") || strings.Contains(item.Name, "devbar") {
			t.Fatalf("CLI export included development file %q", item.Name)
		}
	}
	if !foundIndex {
		t.Fatal("CLI export did not include index.html")
	}
}

func TestTranslateCommandTranslatesAWholeMixedFormatSite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "translated-docs")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	configurationPath := filepath.Join(root, "mpress.yaml")
	configuration, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatal(err)
	}
	configuration = []byte(strings.Replace(string(configuration), "  provider: openrouter", "  provider: codex", 1))

	helperSource := filepath.Join(t.TempDir(), "provider.go")
	helperBinary := filepath.Join(t.TempDir(), "translation-provider")
	helper := `package main
import (
  "bufio"
  "encoding/json"
  "os"
  "strings"
)
func main() {
  scanner := bufio.NewScanner(os.Stdin)
  scanner.Buffer(make([]byte, 1024), 4<<20)
  var request struct { Segments []struct { ID string ` + "`json:\"id\"`" + `; Text string ` + "`json:\"text\"`" + ` } ` + "`json:\"segments\"`" + ` }
  for scanner.Scan() {
    line := strings.TrimSpace(scanner.Text())
    if strings.HasPrefix(line, "{") { _ = json.Unmarshal([]byte(line), &request) }
  }
  response := struct { Translations []struct { ID string ` + "`json:\"id\"`" + `; Text string ` + "`json:\"text\"`" + ` } ` + "`json:\"translations\"`" + ` }{}
  for _, segment := range request.Segments {
    response.Translations = append(response.Translations, struct { ID string ` + "`json:\"id\"`" + `; Text string ` + "`json:\"text\"`" + ` }{segment.ID, "FR " + segment.Text})
  }
  _ = json.NewEncoder(os.Stdout).Encode(response)
}`
	if err := os.WriteFile(helperSource, []byte(helper), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("go", "build", "-o", helperBinary, helperSource).CombinedOutput(); err != nil {
		t.Fatalf("build translation provider: %v\n%s", err, output)
	}
	configuration = append(configuration, []byte("  command: "+helperBinary+"\n")...)
	// command belongs to translation, not deploy. Move it immediately after the
	// provider so YAML scoping remains explicit.
	configuration = []byte(strings.Replace(string(configuration), "  provider: codex\n", "  provider: codex\n  command: "+helperBinary+"\n", 1))
	configuration = []byte(strings.TrimSuffix(string(configuration), "  command: "+helperBinary+"\n"))
	if err := os.WriteFile(configurationPath, configuration, 0o644); err != nil {
		t.Fatal(err)
	}

	mpdPage := `---
schema = 1
title = "Native guide"
translationKey = "native-guide"
---

# Native documentation

@note type="info" title="Before you begin"
Read the instructions.
@end

@steps
@step title="Build"
Run ` + "`mpress build`" + `.
@end
@end
`
	if err := os.WriteFile(filepath.Join(root, "content", "native.mpd"), []byte(mpdPage), 0o644); err != nil {
		t.Fatal(err)
	}
	navigation := "- label: Home\n  link: /\n- label: Native guide\n  link: /native/\n"
	if err := os.WriteFile(filepath.Join(root, "content", "_nav.yaml"), []byte(navigation), 0o644); err != nil {
		t.Fatal(err)
	}

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := translateSite([]string{"--lang", "fr", "--add-language", "--label", "Français"}); err != nil {
		t.Fatal(err)
	}
	if err := translateSite([]string{"--lang", "FR", "--add-language", "--label", "Français"}); err != nil {
		t.Fatalf("resuming an already configured language: %v", err)
	}
	translatedConfig, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(translatedConfig.Site.Languages) != 2 || translatedConfig.Site.Languages[1] != "fr" || translatedConfig.Site.LanguageLabels["fr"] != "Français" {
		t.Fatalf("CLI did not persist the new language: %#v", translatedConfig.Site)
	}

	for _, name := range []string{"index.md", "getting-started.md", "components.md", "native.mpd", "_nav.yaml"} {
		path := filepath.Join(root, "content", "fr", name)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("whole-site translation did not create %s: %v", name, readErr)
			continue
		}
		if !strings.Contains(string(data), "FR ") {
			t.Errorf("translated file %s contains no provider output", name)
		}
	}
	if result, err := site.Build(root, site.BuildOptions{Strict: true}); err != nil {
		t.Fatalf("translated site does not build strictly: %v; diagnostics: %#v", err, result.Diagnostics)
	}
	for _, output := range []string{"fr/index.html", "fr/getting-started/index.html", "fr/components/index.html", "fr/native/index.html"} {
		if _, err := os.Stat(filepath.Join(root, "site", filepath.FromSlash(output))); err != nil {
			t.Errorf("translated route %s was not generated: %v", output, err)
		}
	}
}

func TestCanonicalLanguageCode(t *testing.T) {
	for input, want := range map[string]string{
		"FR":      "fr",
		"pt-br":   "pt-BR",
		"ZH-hant": "zh-Hant",
		"es-419":  "es-419",
	} {
		got, err := canonicalLanguageCode(input)
		if err != nil {
			t.Fatalf("canonicalLanguageCode(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("canonicalLanguageCode(%q) = %q, want %q", input, got, want)
		}
	}
	for _, input := range []string{"", "-fr", "fr-", "fr--CA", "fr_CA"} {
		if _, err := canonicalLanguageCode(input); err == nil {
			t.Errorf("canonicalLanguageCode(%q) succeeded", input)
		}
	}
}

func TestDryRunLanguageAdditionDoesNotWriteConfiguration(t *testing.T) {
	root := t.TempDir()
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "mpress.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	language, changed, err := addTranslationLanguage(root, &cfg, "PT-br", "Português (Brasil)", false)
	if err != nil {
		t.Fatal(err)
	}
	if language != "pt-BR" || !changed {
		t.Fatalf("unexpected preview result: language=%q changed=%v", language, changed)
	}
	if got := cfg.Site.LanguageLabels[language]; got != "Português (Brasil)" {
		t.Fatalf("in-memory preview label = %q", got)
	}
	after, err := os.ReadFile(filepath.Join(root, "mpress.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("dry-run language addition wrote mpress.yaml")
	}
}

func TestTranslateEstimateDoesNotContactProviderOrWriteFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "estimated-site")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	configurationPath := filepath.Join(root, "mpress.yaml")
	before, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := translateSite([]string{"--lang", "zh-CN", "--add-language", "--label", "简体中文", "--harness", "claudecode", "--model", "claude-opus-5", "--estimate", "--json"}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("--estimate changed mpress.yaml")
	}
	if _, err := os.Stat(filepath.Join(root, "content", "zh-CN")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("--estimate wrote a target language directory: %v", err)
	}
}

func TestApplyTranslationHarness(t *testing.T) {
	for name, want := range map[string]struct {
		provider string
		command  string
		model    string
	}{
		"codex":       {provider: "codex", command: "codex", model: "gpt-5.6-sol"},
		"claudecode":  {provider: "claude", command: "claude", model: "claude-opus-4.6"},
		"claude-code": {provider: "claude", command: "claude", model: "claude-opus-4.6"},
		"claude":      {provider: "claude", command: "claude", model: "claude-opus-4.6"},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Translation.Command = "configured-provider"
			if err := applyTranslationHarness(&cfg, name); err != nil {
				t.Fatal(err)
			}
			if cfg.Translation.Provider != want.provider || cfg.Translation.Command != want.command || cfg.Translation.Model != want.model {
				t.Fatalf("harness %q configured %#v", name, cfg.Translation)
			}
		})
	}
	cfg := config.Default()
	if err := applyTranslationHarness(&cfg, "unknown"); err == nil || !strings.Contains(err.Error(), "codex or claudecode") {
		t.Fatalf("unknown harness was accepted: %v", err)
	}
}

func TestConvertCommandReplacesCompleteMarkdownSiteWithMPD(t *testing.T) {
	root := filepath.Join(t.TempDir(), "converted-site")
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := convertSite([]string{"--replace"}); err != nil {
		t.Fatal(err)
	}
	var markdown, documents int
	err = filepath.WalkDir(filepath.Join(root, "content"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md", ".markdown":
			markdown++
		case ".mpd":
			documents++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if markdown != 0 || documents < 3 {
		t.Fatalf("converted corpus has markdown=%d mpd=%d", markdown, documents)
	}
	if result, err := site.Build(root, site.BuildOptions{Strict: true}); err != nil {
		t.Fatalf("converted MPD site does not build strictly: %v; diagnostics: %#v", err, result.Diagnostics)
	}
	if err := convertSite([]string{"--to", "markdown", "--replace"}); err != nil {
		t.Fatalf("convert MPD site back to Markdown: %v", err)
	}
	markdown, documents = 0, 0
	err = filepath.WalkDir(filepath.Join(root, "content"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md", ".markdown":
			markdown++
		case ".mpd":
			documents++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if documents != 0 || markdown < 3 {
		t.Fatalf("round-tripped corpus has markdown=%d mpd=%d", markdown, documents)
	}
	if result, err := site.Build(root, site.BuildOptions{Strict: true}); err != nil {
		t.Fatalf("round-tripped Markdown site does not build strictly: %v; diagnostics: %#v", err, result.Diagnostics)
	}
}
