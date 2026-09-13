package dev

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func TestEditorCommandPrefersConfiguredEditor(t *testing.T) {
	t.Setenv("VISUAL", "codium --reuse-window")
	t.Setenv("EDITOR", "")
	command, label, err := editorCommand("/tmp/guide.md", func(name string) (string, error) {
		if name == "codium" {
			return "/usr/bin/codium", nil
		}
		return "", os.ErrNotExist
	})
	if err != nil {
		t.Fatal(err)
	}
	if label != "codium" || strings.Join(command.Args, " ") != "/usr/bin/codium --reuse-window /tmp/guide.md" {
		t.Fatalf("unexpected editor command: %q %#v", label, command.Args)
	}
}

func (t bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}

func TestMCPShipsWithDevServerAndRequiresServerBearerToken(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "server-secret", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/__mpress/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized || !strings.HasPrefix(response.Header.Get("WWW-Authenticate"), "Bearer") {
		t.Fatalf("MCP request without bearer token returned %d and %q", response.StatusCode, response.Header.Get("WWW-Authenticate"))
	}
	response.Body.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "mpress-test", Version: "test"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL + "/__mpress/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: httpServer.Client().Transport, token: "server-secret"}},
		DisableStandaloneSSE: true,
	}
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range listed.Tools {
		names[tool.Name] = true
	}
	for _, name := range []string{"project_info", "list_files", "read_file", "write_file", "get_config", "update_config", "check_site", "list_versions", "capture_version", "deploy_site"} {
		if !names[name] {
			t.Errorf("MCP server is missing %q", name)
		}
	}

	file := getFile(t, httpServer.URL, "content/index.md")
	updated := strings.Replace(file.Content, "Original **paragraph**", "Changed through MCP", 1)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "write_file", Arguments: map[string]any{"path": file.Path, "content": updated, "revision": file.Revision}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("MCP write failed: %#v", result.Content)
	}
	written, err := os.ReadFile(filepath.Join(root, "content", "index.md"))
	if err != nil || !strings.Contains(string(written), "Changed through MCP") {
		t.Fatalf("MCP did not persist the file: %v, %s", err, written)
	}
}

func TestAuthoringAPIReadsWritesBacksUpAndRebuildsMarkdown(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	file := getFile(t, httpServer.URL, "content/index.md")
	if file.Route != "/" || !strings.Contains(file.Content, "Original **paragraph**") {
		t.Fatalf("unexpected source payload: %#v", file)
	}
	updated := strings.Replace(file.Content, "Original **paragraph**", "Updated paragraph", 1)
	response := requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/file", filePayload{Path: file.Path, Content: updated, Revision: file.Revision}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("write status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	output, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil || !strings.Contains(string(output), "Updated paragraph") {
		t.Fatalf("rebuilt output did not contain update: %v, %s", err, output)
	}
	backups, err := filepath.Glob(filepath.Join(root, ".mpress", "backups", "*", "content", "index.md"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one recoverable backup, got %v, %v", backups, err)
	}
	backup, _ := os.ReadFile(backups[0])
	if !strings.Contains(string(backup), "Original **paragraph**") {
		t.Fatalf("backup lost original content: %s", backup)
	}
}

func TestKnowledgeWorkspaceReportsHealthAndSearchesCurrentBuild(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/__mpress/api/knowledge?q=editable+paragraph")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("knowledge status=%d: %s", response.StatusCode, readBody(response))
	}
	var payload struct {
		Enabled  bool `json:"enabled"`
		Ready    bool `json:"ready"`
		Pages    int  `json:"pages"`
		Sections int  `json:"sections"`
		Results  []struct {
			URL string `json:"url"`
		} `json:"results"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Enabled || !payload.Ready || payload.Pages != 1 || payload.Sections == 0 || len(payload.Results) == 0 || payload.Results[0].URL != "/#editable-heading" {
		t.Fatalf("unexpected knowledge health: %#v", payload)
	}
	if !strings.Contains(devbarJS, "Knowledge base") || !strings.Contains(devbarJS, "showKnowledge") {
		t.Fatal("development workspace does not expose the knowledge base")
	}
}

func TestLighthouseAPIRejectsExternalURLs(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/lighthouse", map[string]string{"path": "https://example.com/", "formFactor": "mobile"}, "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("external Lighthouse URL returned %d: %s", response.StatusCode, readBody(response))
	}
}

func TestAuthoringAPIRejectsTraversalStaleWritesAndRemoteWritesWithoutToken(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/__mpress/api/file?path=../private.txt")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("traversal status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	file := getFile(t, httpServer.URL, "content/index.md")
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/file", filePayload{Path: file.Path, Content: file.Content + "\nChanged", Revision: file.Revision}, "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("remote write without token status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/preview", previewPayload{Path: file.Path, Content: file.Content + "\nPreview", Revision: file.Revision}, "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("remote preview without token status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/file", filePayload{Path: file.Path, Content: file.Content + "\nChanged", Revision: "stale"}, "secret")
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("stale write status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
}

func TestAuthoringAPIFindsEditableBlocksAndPersistsOneSourceRange(t *testing.T) {
	root := authoringFixture(t)
	server, _ := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/__mpress/api/blocks?path=content/index.md")
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Revision string          `json:"revision"`
		Blocks   []editableBlock `json:"blocks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if len(payload.Blocks) < 2 || payload.Blocks[0].Kind != "heading" || payload.Blocks[1].Text != "Original paragraph with emphasis." {
		t.Fatalf("unexpected editable blocks: %#v", payload.Blocks)
	}
	block := payload.Blocks[1]
	write := blockPayload{Path: "content/index.md", Start: block.Start, End: block.End, Markdown: "Changed **paragraph** with emphasis.", Revision: payload.Revision}
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/block", write, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("block write status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	data, _ := os.ReadFile(filepath.Join(root, "content", "index.md"))
	if !strings.Contains(string(data), "Changed **paragraph**") {
		t.Fatalf("block update did not preserve Markdown: %s", data)
	}
}

func TestAuthoringAPIImportsMarkdownAndWorkspaceIsAnOptionalExtension(t *testing.T) {
	root := authoringFixture(t)
	server, _ := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "new-guide.md")
	_, _ = io.WriteString(part, "---\ntitle: New guide\n---\n\nImported Markdown.\n")
	_ = writer.Close()
	request, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/__mpress/api/import", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("import status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	if _, err := os.Stat(filepath.Join(root, "content", "new-guide.md")); err != nil {
		t.Fatal(err)
	}

	response, err = http.Get(httpServer.URL + "/__mpress/")
	if err != nil {
		t.Fatal(err)
	}
	markup := readBody(response)
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("development site permits stale generated assets: Cache-Control=%q", got)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(markup, "data-mpress-workspace") || !strings.Contains(markup, "/__mpress/devbar.js") || !strings.Contains(markup, `<base href="/">`) {
		t.Fatalf("open-source project workspace was not served: %d %s", response.StatusCode, markup)
	}
	for _, want := range []string{"mpress-workspace-boot-style", "body:not(.mpress-workspace-document)"} {
		if !strings.Contains(markup, want) {
			t.Errorf("workspace response is missing its no-flash bootstrap %q", want)
		}
	}
	baseIndex := strings.Index(markup, `<base href="/">`)
	stylesheetIndex := strings.Index(markup, `href="./assets/mpress.css?v=`)
	if stylesheetIndex < 0 || baseIndex > stylesheetIndex {
		t.Fatalf("workspace base URL must precede route-relative stylesheet URLs: %s", markup)
	}
	if strings.Contains(markup, "/__mpress/app.js") {
		t.Fatalf("open-source project workspace unexpectedly includes a private extension: %s", markup)
	}

	workspace := &WorkspaceAssets{
		HTML: "<!doctype html><title>Optional workspace</title>{{icon:save}}",
		CSS:  ".workspace { display: grid; }",
		JS:   "window.optionalWorkspace = true;",
	}
	extended, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true, Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	extendedHTTP := httptest.NewServer(extended.Handler())
	defer extendedHTTP.Close()
	response, err = http.Get(extendedHTTP.URL + "/__mpress/")
	if err != nil {
		t.Fatal(err)
	}
	markup = readBody(response)
	response.Body.Close()
	if !strings.Contains(markup, "Optional workspace") || !strings.Contains(markup, "lucide-save") {
		t.Fatalf("optional workspace was not rendered through the extension: %s", markup)
	}
	response, err = http.Get(extendedHTTP.URL + "/__mpress/app.js")
	if err != nil {
		t.Fatal(err)
	}
	script := readBody(response)
	response.Body.Close()
	if script != workspace.JS {
		t.Fatalf("optional workspace script mismatch: %q", script)
	}
}

func TestGeneratedSiteGetsWorkingDevelopmentBarWithoutChangingBuildOutput(t *testing.T) {
	root := authoringFixture(t)
	contribution := &ContributionSession{SiteURL: "https://docs.example.test/guide/", Repository: "https://github.com/example/docs.git", SourcePath: "content/guide.md", Route: "/guide/", Branch: "contribute/20260806-153000"}
	server, _ := NewServer(root, Options{Host: "127.0.0.1", Authoring: true, Contribution: contribution})
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	projectResponse, err := http.Get(httpServer.URL + "/__mpress/api/project")
	if err != nil {
		t.Fatal(err)
	}
	var project map[string]any
	if err := json.NewDecoder(projectResponse.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}
	projectResponse.Body.Close()
	if project["token"] != server.token {
		t.Fatal("a loopback dev page did not receive the current authoring token")
	}
	contributionPayload, ok := project["contribution"].(map[string]any)
	if !ok || contributionPayload["sourcePath"] != "content/guide.md" || contributionPayload["branch"] != "contribute/20260806-153000" {
		t.Fatalf("contribution session missing from project API: %#v", project["contribution"])
	}
	remoteRequest := httptest.NewRequest(http.MethodGet, "/__mpress/api/project", nil)
	remoteRequest.RemoteAddr = "203.0.113.12:54321"
	remoteResponse := httptest.NewRecorder()
	server.handleProject(remoteResponse, remoteRequest)
	var remoteProject map[string]any
	if err := json.Unmarshal(remoteResponse.Body.Bytes(), &remoteProject); err != nil {
		t.Fatal(err)
	}
	if _, exposed := remoteProject["token"]; exposed {
		t.Fatal("the authoring token was exposed to a remote unpaired browser")
	}

	response, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	markup := readBody(response)
	response.Body.Close()
	for _, want := range []string{"mpress-devbar-host", "/__mpress/devbar.js", "--mpress-devbar-height:42px", "padding-bottom:42px", "scroll-padding-bottom:58px", ".mpress-dev-edit-site", ".mpress-dev-image-edit", "top:12px;right:12px", ".mpress-dev-image-resize", "width:24px;height:44px", "inset:7px 8px", "width:10px;height:10px", "cursor:ew-resize", ".mpress-dev-image-menu", "top:56px;right:12px", ".mpress-dev-image-background", ".mpress-dev-image-panel-controls"} {
		if !strings.Contains(markup, want) {
			t.Errorf("served site missing %q", want)
		}
	}
	auditRequest, err := http.NewRequest(http.MethodGet, httpServer.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	auditRequest.Header.Set("X-MPress-Audit", "lighthouse")
	auditResponse, err := http.DefaultClient.Do(auditRequest)
	if err != nil {
		t.Fatal(err)
	}
	auditMarkup := readBody(auditResponse)
	auditResponse.Body.Close()
	if strings.Contains(auditMarkup, "mpress-devbar-host") || strings.Contains(auditMarkup, "/__mpress/devbar.js") {
		t.Fatal("Lighthouse audit response contains development UI")
	}
	previewResponse, err := http.Get(httpServer.URL + "/?__mpress_preview=1")
	if err != nil {
		t.Fatal(err)
	}
	previewMarkup := readBody(previewResponse)
	previewResponse.Body.Close()
	if strings.Contains(previewMarkup, "mpress-devbar-host") || strings.Contains(previewMarkup, "/__mpress/devbar.js") {
		t.Fatal("responsive iframe response contains nested development UI")
	}
	built, _ := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if strings.Contains(string(built), "mpress-devbar-host") {
		t.Fatal("development bar leaked into static build output")
	}
	response, err = http.Get(httpServer.URL + "/__mpress/devbar.html")
	if err != nil {
		t.Fatal(err)
	}
	bar := readBody(response)
	response.Body.Close()
	for _, want := range []string{"Back to site", "Settings", "Run checks", "Lighthouse audit", "Export ZIP", "Translations", "Deploy", "Project guide", "lucide-pencil", "lucide-gauge", "lucide-package", "lucide-languages", "lucide-rocket", "lucide-smartphone", "lucide-tablet", "lucide-monitor", "lucide-bold", "lucide-italic", "lucide-code", "lucide-file-text", "lucide-eye", `id="icon-list"`, `id="icon-list-ordered"`, `id="icon-quote"`, "data-preview-size=\"mobile\"", "data-preview-size=\"tablet\"", "data-preview-size=\"web\"", "viewBox=\"0 0 240 240\"", "fill-rule=\"evenodd\"", "aria-label=\"Open M-Press project editor\"", "aria-label=\"Run project checks\""} {
		if !strings.Contains(bar, want) {
			t.Errorf("development menu missing %q", want)
		}
	}
	if strings.Contains(bar, `{{brand:mpress-mark}}`) || !strings.Contains(bar, `<span class="mark" aria-hidden="true"><svg`) {
		t.Fatal("development launcher does not render the M-Press logo")
	}
	if strings.Contains(bar, `class="launcher-label">Edit site</span>`) {
		t.Fatal("development launcher still includes the removed Edit site label")
	}
	for _, excluded := range []string{"Open Studio", "Edit site", "installSiteEditorEntry"} {
		if strings.Contains(bar, excluded) {
			t.Errorf("open-source development menu contains private Studio action %q", excluded)
		}
	}
	response, err = http.Get(httpServer.URL + "/__mpress/devbar.js")
	if err != nil {
		t.Fatal(err)
	}
	barScript := readBody(response)
	response.Body.Close()
	if !strings.Contains(barScript, "events.addEventListener('reload'") || !strings.Contains(barScript, "location.reload()") {
		t.Fatal("generated development pages do not reload after a source rebuild")
	}
	for _, want := range []string{"openWorkspace", "workspaceMode", "prepareWorkspaceDocument", "installWorkspaceSidebar", "workspaceConfigPages", "workspaceNavigationGroups", "mpress-workspace-sidebar", "mpress-workspace-group-toggle", "mpress-workspace-group-links", "data-workspace-group-toggle", "mpress-workspace-nav:", "data-workspace-command", "translations", "checks", "lighthouse", "publish", "config-blog", "showPublish", "Download a production ZIP", "Cloudflare Pages", "Netlify", "publish-providers", "provider-logo", "Project editor", "Revert changes", "Save and rebuild", "config-reset-guided", "config-reset-complete", "config-form?section=${section}", "formSections", "settingsFormTemplates", "mpress-settings-form-fields", "setSiteValue", "setSettingsValue", "setSettingsChecked", "installShortcutCapture", "data-shortcut-change", "Change key", "Press key combination", "shortcutCaptureInstalled", "installStyledSelects", "mpress-select-trigger", "mpress-select-option", "normaliseShortcutValue", "Conflicts with", "All settings", `aria-label="Configuration sections"`, `data-settings-tab="site"`, `data-settings-tab="blog"`, `data-settings-tab="accessibility"`, `data-settings-tab="layout"`, `data-settings-tab="translation"`, `data-settings-tab="deploy"`, `data-settings-page="site"`, `data-settings-page="blog"`, `data-settings-page="accessibility"`, `data-settings-page="layout"`, `data-settings-page="translation"`, `data-settings-page="deploy"`, "settings-previous", "settings-next", `name="search.shortcut"`, `name="accessibility.shortcut"`, `complete.search.shortcut`, `complete.accessibility.shortcut`, "translation.inputPricePerMillion", "translation.outputPricePerMillion", "blog.showTags", "data-deploy-domain", "add-language", "add-header-link", "add-deploy-target", "showTranslations", "open-translation-workspace", "ensureRepositoryReady", "showRepositorySetup", "repository?.onWorkBranch", "Copy command", "add-language", "Automatic setup is ready", "Compare models", "Run blind comparison", "Choose the better translation", "Review translation plan", "Quality refinement", "Human approval", "What would you like to improve?", "showContributionWizard", "showContributionPage", "mpress-contribution-wizard", "blog-posts", "markdown-preview", "version-create", "version-status", "settings-notice"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("complete configuration form is missing %q", want)
		}
	}
	for _, want := range []string{"blog.landingStyle", "blog.tagline", "blog-new", "blog-posts", "blog-preferences", "data-open-blog-post", `aria-label="Open ${escapeHTML(post.title)}"`, "Blog post view", "blog-post-preview", `data-blog-post-mode="view"`, `data-blog-post-mode="edit"`, "replaceChildren(article)", "blog-post-preview-document", "data-blog-tag-editor", "data-blog-tag-palette", "data-add-blog-tag", "data-remove-blog-tag", `data-blog-editor-tab="markdown"`, `data-blog-editor-tab="write"`, `data-blog-editor-tab="details"`, "blog-editor-details", "markdown-toolbar", "data-markdown-format", "data-markdown-count", "Use Tab to indent", "Press ${/(Mac|iPhone|iPad|iPod)/i.test", "insertMarkdown", "contentEditable = 'true'", "richTextToMarkdown", "syncRichTextToMarkdown", "data-rich-format", "data-rich-block-format", "renderRichTextEditor", "Formatting is saved as Markdown", "The visual editor is temporarily unavailable", "document.execCommand", "Could not copy the code", "navigator.clipboard.writeText", "Discard unsaved changes to this post?", "workspaceHasUnsavedChanges"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("blog workspace is missing %q", want)
		}
	}
	for _, want := range []string{"image-asset", "openImageEditor", "data-image-crop", "data-image-brightness", "data-image-text", "data-image-overwrite", "site.logoLightUpload", "site.logoDarkUpload", "site.faviconUpload", "site.socialImageUpload", "Generate from site", "new FormData", "Logos and images"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("brand settings are missing %q", want)
		}
	}
	for _, want := range []string{"openContentImageDialog", "installContentImageDrop", "contentImageFiles", "clipboardData?.files", `data-content-image-mode="theme"`, `data-content-image-slot="light"`, `data-content-image-slot="dark"`, "Continue to edit", "This image is decorative", "category: 'content'", `@image{light="`, "classifyThemeImages", "image.draggable = false", "syncDevbarTheme"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("blog editor image workflow is missing %q", want)
		}
	}
	for _, want := range []string{"language-table", "language-table-head", "language-cell-label", "language-remove", `aria-label="Configured languages"`, "header-link-table", "header-link-table-head", `aria-label="Header links"`, "iconHTML('x')"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("language settings row editor is missing %q", want)
		}
	}
	for _, excluded := range []string{"enhanceLanguageMenu", "data-mpress-language-action", "mpress-authoring-actions"} {
		if strings.Contains(barScript, excluded) {
			t.Errorf("language selector contains development-only translation action %q", excluded)
		}
	}
	for _, want := range []string{`a.mpress-button[href="#your-first-three-steps"]`, "mpress-onboarding-dismissed:", "openWorkspace('onboarding')", "Choose your starting point", "Keep the guided tutorial", "Start with a minimal site", "Import Markdown files", "wizard-import-guidance", "Continue to validation", "replaceStarter", "action: 'starter'", "[1,2,3,4].map", "Math.max(1, Math.min(4, step))", "Step 4 of 4", "check-performance-details"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("first-run onboarding is missing %q", want)
		}
	}
	for _, want := range []string{"showCurrentPage", "Edit this page", "Copy file path", "state.project.authoring", "document.querySelectorAll('[data-mpress-contribute]')", "showContributionReview", "Review changes", "Commit changes", "Push contribution branch", "Open draft pull request", "contributorGuide", "Source diff", "mpress-contribution-flow", "renderContributionDiff", "contributorGuideHTML", "Contribution guide", "navigationGroups = [{label: 'Contribution'", "savedContributionFlow", "flow !== 'explore'"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("authoring journey is missing %q", want)
		}
	}
	for _, want := range []string{"enhanceBlogImages", "[data-blog-style-editor]", "api('blog-image'", "commitImageDraft", "commitImageWidth", "keepalive: true", "data-image-mode=\"panel\"", "data-image-mode=\"floating\"", "draftMode", "data-image-fit=\"cover\"", "data-tags-visible=\"true\"", "data-heading-size=\"large\"", "mpress-dev-image-resize", "setPointerCapture", "--blog-floating-image-width", "defaultBackground", "#000000", "#ffffff", "state.project.token", "suppressReloadUntil", `type="color"`, "Image appearance saved", "Image size saved", "data-image-background-reset"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("development blog appearance editor is missing %q", want)
		}
	}
	for _, want := range []string{"renderCheckPerformance", "Validation performance", "check-timeline-row", "formatCheckDuration", "result.performance", "Back to guide", "runChecks(true, () => wizard(4))"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("development checks are missing performance chart contract %q", want)
		}
	}
	for _, want := range []string{"showLighthouse", "api('lighthouse'", "Lighthouse audit", "lighthouse-scores", "lighthouse-metrics", "formFactor"} {
		if !strings.Contains(barScript, want) && !strings.Contains(bar, want) {
			t.Errorf("development Lighthouse audit is missing %q", want)
		}
	}
	for _, want := range []string{"showExport", "/__mpress/api/export", "Build and download ZIP", "response.blob()", "X-MPress-Filename", "URL.createObjectURL", "Production ZIP downloaded"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("development export menu is missing %q", want)
		}
	}
	for _, unwanted := range []string{"mpress-dev-image-save-hint", "Preview changes", "Click away to save", "Previewing"} {
		if strings.Contains(barScript, unwanted) || strings.Contains(markup, unwanted) {
			t.Errorf("development blog appearance editor still contains removed message %q", unwanted)
		}
	}
	for _, want := range []string{"lockPageScroll", "unlockPageScroll", "document.documentElement.style.overflow = 'hidden'", "document.body.style.overflow = 'hidden'"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("development drawer is missing page scroll lock %q", want)
		}
	}
	response, err = http.Get(httpServer.URL + "/__mpress/devbar.css")
	if err != nil {
		t.Fatal(err)
	}
	barStyles := readBody(response)
	response.Body.Close()
	for _, unwanted := range []string{"theme-navbar-preview", "theme-preview-panel", "updateThemeNavbarPreview", "Navbar preview", "Light-mode navbar preview", "Dark-mode navbar preview"} {
		if strings.Contains(barScript, unwanted) || strings.Contains(barStyles, unwanted) {
			t.Errorf("appearance settings still contain duplicate navbar preview %q", unwanted)
		}
	}
	for _, want := range []string{`:host([data-theme="light"])`, `--mp-bg: var(--surface-solid, #ffffff);`, `background: var(--mp-bg);`, `.launcher { min-width: 52px; padding-inline: 13px;`, `.mark { display: inline-grid; width: 24px; height: 22px; place-items: center; color: var(--mp-muted); }`, `.mark > svg { display: block; width: 24px; height: 21px; }`, `:host([data-mpress-workspace]) .command-menu { display: none !important; }`, `:host([data-mpress-workspace]) .drawer`, `var(--mp-workspace-sidebar-width, 18.5rem)`, `.workspace-page-header h1`, `grid-template-columns: 1fr`, `.settings-section-title`, `.settings-notice`, `.mpress-settings-form-fields select { appearance: none;`, ".settings-subnav", "overscroll-behavior: contain", "scrollbar-gutter: stable"} {
		if !strings.Contains(barStyles, want) {
			t.Errorf("configuration workspace layout is missing %q", want)
		}
	}
	for _, want := range []string{"syncDevbarTheme", "host.dataset.theme", "attributeFilter: ['data-theme']", "themeMedia.addEventListener"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("development bar theme synchronisation is missing %q", want)
		}
	}
	for _, want := range []string{".markdown-preview img", ".markdown-preview .mpress-terminal-title", ".markdown-preview .mpress-copy", ".rich-text-editor:focus-within", ".markdown-preview.rich-text-surface", ".rich-text-toolbar", ".rich-text-surface blockquote", "max-height: 520px", ".content-image-drop-active::after", ".content-image-dialog", ".content-image-dropzone", ".content-image-pair", `:host([data-theme="dark"]) .rich-text-surface .mpress-theme-image-dark`, ".image-editor-overlay { position: fixed; z-index: 1000; inset: 0; display: grid; place-items: center; padding: 28px; pointer-events: auto;"} {
		if !strings.Contains(barStyles, want) {
			t.Errorf("blog Markdown preview styles are missing %q", want)
		}
	}
	if strings.Contains(barStyles, "box-shadow: inset 3px 0 var(--mp-accent)") {
		t.Fatal("the Markdown editor still uses the removed left-edge focus highlight")
	}
	for _, want := range []string{"beginConfigPreview", "applyConfigPreview", "restoreConfigPreview", "workspaceAccent", "config-reset-guided", "config-reset-complete", "q('#scrim').hidden = floating", "installColorControls", "mpress-color-reset", "six-digit hexadecimal colour"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("configuration live-preview overlay is missing %q", want)
		}
	}
	for _, want := range []string{"Automatic setup is ready", "translation-pipeline", "renderAddLanguage", "version-create-error"} {
		if !strings.Contains(barScript, want) {
			t.Errorf("content workflow polish is missing %q", want)
		}
	}
}

func TestDevelopmentExportDownloadsAProductionZIP(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "content/draft.md", "---\ntitle: Draft\ndraft: true\n---\n\nDevelopment only.\n")
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/export", map[string]bool{"strict": true, "drafts": false}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("export status %d: %s", response.StatusCode, readBody(response))
	}
	if response.Header.Get("Content-Type") != "application/zip" || !strings.Contains(response.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("unexpected export headers: %#v", response.Header)
	}
	data, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	foundIndex := false
	for _, item := range archive.File {
		if item.Name == "index.html" {
			foundIndex = true
			reader, openErr := item.Open()
			if openErr != nil {
				t.Fatal(openErr)
			}
			markup, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			if strings.Contains(string(markup), "mpress-devbar-host") {
				t.Fatal("development bar leaked into exported HTML")
			}
		}
		if item.Name == "draft/index.html" {
			t.Fatal("development export included a draft page")
		}
	}
	if !foundIndex {
		t.Fatal("development export did not contain index.html")
	}
}

func TestDevelopmentBlogImageEditorPersistsImageFitInFrontmatter(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "content/blog/index.md", "---\ntitle: Blog\n---\n\nBlog archive.\n")
	writeDevFixture(t, root, "content/blog/latest.md", "---\ntitle: Latest post\ndate: 2026-08-02\nimage: /images/latest.png\n---\n\nLatest article.\n")
	writeDevFixture(t, root, "content/_nav.yaml", "- label: Home\n  link: /\n- label: Blog\n  link: /blog/\n")
	writeDevFixture(t, root, "static/images/latest.png", "image")
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/blog/")
	if err != nil {
		t.Fatal(err)
	}
	markup := readBody(response)
	response.Body.Close()
	for _, want := range []string{`data-blog-style-editor`, `data-blog-image-editor`, `data-blog-image-mode="panel"`, `data-blog-image-fit="cover"`, `data-blog-image-width="100"`, `data-blog-tags-visible="true"`, `data-blog-heading-size="default"`, `mpress-dev-image-edit`, `mpress-dev-image-menu`} {
		if !strings.Contains(markup, want) {
			t.Errorf("served blog is missing development appearance editing support %q", want)
		}
	}

	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Fit: "contain"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("image fit update status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, err := os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "imageFit: contain") || !strings.Contains(string(source), "Latest article.") {
		t.Fatalf("image fit update damaged the Markdown source: %s", source)
	}
	server.rebuild(false)
	built, err := os.ReadFile(filepath.Join(root, "site", "blog", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(built), `data-blog-image-fit="contain"`) {
		t.Fatalf("rebuilt blog did not apply the persisted fit: %s", built)
	}
	background := "#123456"
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Background: &background}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("image background update status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if !strings.Contains(string(source), "imageBackground:") || !strings.Contains(string(source), "#123456") {
		t.Fatalf("image background was not saved in frontmatter: %s", source)
	}
	server.rebuild(false)
	built, _ = os.ReadFile(filepath.Join(root, "site", "blog", "index.html"))
	if !strings.Contains(string(built), `data-blog-image-background="#123456"`) || !strings.Contains(string(built), `--blog-image-background:#123456`) {
		t.Fatalf("rebuilt blog did not apply the image background: %s", built)
	}
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Mode: "floating"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("floating image mode status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if !strings.Contains(string(source), "imageMode: floating") || strings.Contains(string(source), "imageBackground:") {
		t.Fatalf("floating image mode was not saved without a background: %s", source)
	}
	server.rebuild(false)
	built, _ = os.ReadFile(filepath.Join(root, "site", "blog", "index.html"))
	if !strings.Contains(string(built), `data-blog-image-mode="floating"`) || strings.Contains(string(built), `data-blog-image-background=`) {
		t.Fatalf("rebuilt blog did not apply the floating image mode: %s", built)
	}
	width := 74
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Width: &width}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("floating image width status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if !strings.Contains(string(source), "imageWidth: 74") {
		t.Fatalf("floating image width was not saved in frontmatter: %s", source)
	}
	server.rebuild(false)
	built, _ = os.ReadFile(filepath.Join(root, "site", "blog", "index.html"))
	if !strings.Contains(string(built), `data-blog-image-width="74"`) || !strings.Contains(string(built), `--blog-floating-image-width:74%`) {
		t.Fatalf("rebuilt blog did not apply the floating image width: %s", built)
	}
	width = 100
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Width: &width}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("floating image width reset status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if strings.Contains(string(source), "imageWidth:") {
		t.Fatalf("default floating image width left frontmatter behind: %s", source)
	}
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Mode: "panel"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("panel image mode status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if strings.Contains(string(source), "imageMode:") {
		t.Fatalf("default panel image mode left frontmatter behind: %s", source)
	}
	background = ""
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Background: &background}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("image background reset status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if strings.Contains(string(source), "imageBackground:") {
		t.Fatalf("default image background left an empty frontmatter setting: %s", source)
	}
	showTags := false
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", ShowTags: &showTags, HeadingSize: "large"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("blog presentation update status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if !strings.Contains(string(source), "showTags: false") || !strings.Contains(string(source), "headingSize: large") {
		t.Fatalf("blog presentation settings were not saved in frontmatter: %s", source)
	}
	showTags = true
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", ShowTags: &showTags, HeadingSize: "default"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("blog presentation reset status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	source, _ = os.ReadFile(filepath.Join(root, "content", "blog", "latest.md"))
	if strings.Contains(string(source), "showTags:") || strings.Contains(string(source), "headingSize:") {
		t.Fatalf("default blog presentation settings left frontmatter behind: %s", source)
	}

	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Fit: "stretch"}, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid image fit status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Mode: "detached"}, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid image mode status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", HeadingSize: "huge"}, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid heading size status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	width = 49
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Width: &width}, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid image width status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	background = "red"
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-image", blogImagePayload{Route: "/blog/latest/", Background: &background}, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid image background status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
}

func TestTranslationStatusAPIIsReadOnlyAndTranslationWriteNeedsToken(t *testing.T) {
	root := authoringFixture(t)
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/__mpress/api/translations?lang=fr&file=content/index.md")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status request = %d: %s", response.StatusCode, readBody(response))
	}
	var status struct {
		DefaultLanguage string                     `json:"defaultLanguage"`
		Automatic       map[string]json.RawMessage `json:"automatic"`
		Report          struct {
			Pending int `json:"pending"`
		} `json:"report"`
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if status.DefaultLanguage != "en" || status.Report.Pending == 0 {
		t.Fatalf("unexpected translation status: %#v", status)
	}
	for _, stage := range []string{"ready", "draft", "refinement", "audit"} {
		if _, ok := status.Automatic[stage]; !ok {
			t.Fatalf("translation status is missing automatic %s selection: %#v", stage, status.Automatic)
		}
	}

	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/translations", map[string]any{"language": "fr", "file": "content/index.md"}, "")
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("translation without token = %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
}

func TestLivePreviewBuildsUnsavedMarkdownWithoutChangingSource(t *testing.T) {
	root := authoringFixture(t)
	server, _ := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	file := getFile(t, httpServer.URL, "content/index.md")
	previewSource := strings.Replace(file.Content, "Original **paragraph**", "Unsaved live preview", 1)
	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/preview", previewPayload{Path: file.Path, Content: previewSource, Revision: file.Revision}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("preview status %d: %s", response.StatusCode, readBody(response))
	}
	var payload struct {
		OK    bool   `json:"ok"`
		URL   string `json:"url"`
		Route string `json:"route"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !payload.OK || payload.URL == "" || payload.Route != "/" {
		t.Fatalf("unexpected preview payload: %#v", payload)
	}
	response, err := http.Get(httpServer.URL + payload.URL)
	if err != nil {
		t.Fatal(err)
	}
	preview := readBody(response)
	response.Body.Close()
	if !strings.Contains(preview, "Unsaved live preview") {
		t.Fatalf("preview output does not contain the unsaved edit: %s", preview)
	}
	source, _ := os.ReadFile(filepath.Join(root, "content", "index.md"))
	if !strings.Contains(string(source), "Original **paragraph**") || strings.Contains(string(source), "Unsaved live preview") {
		t.Fatalf("live preview changed the source file: %s", source)
	}
	saved, _ := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if !strings.Contains(string(saved), "Original <strong>paragraph</strong>") || strings.Contains(string(saved), "Unsaved live preview") {
		t.Fatalf("live preview changed the normal build output: %s", saved)
	}

	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/preview", previewPayload{Path: file.Path, Content: "---\ntitle: Broken", Revision: file.Revision}, "")
	var invalid struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(response.Body).Decode(&invalid); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if invalid.OK {
		t.Fatal("invalid Markdown unexpectedly produced a live preview")
	}
	response, err = http.Get(httpServer.URL + payload.URL)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("last valid preview was not preserved: %d", response.StatusCode)
	}
	response.Body.Close()
}

func TestDevelopmentConfigEditingPreservesCommentsAndRebuilds(t *testing.T) {
	root := authoringFixture(t)
	path := filepath.Join(root, "mpress.yaml")
	original, _ := os.ReadFile(path)
	_ = os.WriteFile(path, append([]byte("# Keep this project note\n"), original...), 0o644)
	server, _ := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, _ := http.Get(httpServer.URL + "/__mpress/api/config")
	var payload configPayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	payload.Title = "Changed project"
	payload.Description = "Changed from the generated site"
	payload.ColorScheme = "dark"
	payload.AccentColor = "#3456ef"
	payload.HoverColor = "#89a1ff"
	payload.HoverColorLight = "#223344"
	payload.HoverColorDark = "#ddeeff"
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/config", payload, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("config update status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "# Keep this project note") || !strings.Contains(string(updated), "title: Changed project") {
		t.Fatalf("config update lost content: %s", updated)
	}
	page, _ := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if !strings.Contains(string(page), "Changed project") {
		t.Fatalf("site was not rebuilt with updated config: %s", page)
	}

	response, _ = http.Get(httpServer.URL + "/__mpress/api/config")
	var complete configPayload
	if err := json.NewDecoder(response.Body).Decode(&complete); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if complete.Config == nil || complete.Config.Site.Title != "Changed project" {
		t.Fatalf("complete configuration did not include structured settings: %#v", complete.Config)
	}
	complete.Complete = true
	complete.Config.Search.Enabled = false
	complete.Config.Search.Shortcut = "Control+Shift+K"
	complete.Config.Search.Placeholder = "Find the documentation"
	complete.Config.Search.MaxResults = 18
	complete.Config.Search.RememberRecent = false
	complete.Config.Accessibility.Shortcut = "Mod+A"
	complete.Config.Social.GitHub = "https://example.com/project"
	complete.Config.Contribution = config.ContributionConfig{Enabled: true, Repository: "https://github.com/example/docs.git", Branch: "main"}
	complete.Config.Site.HeaderLinks = []config.HeaderLink{{Label: "Home", URL: "/"}}
	complete.Config.Theme.Layout = config.LayoutConfig{Preset: "custom", ContentWidth: "75%", WideContentWidth: "1100px", SidebarWidth: "300px", TOCWidth: "240px", ContentTOCGap: "32px", Alignment: "cluster", TOC: "right"}
	complete.Config.Blog = config.BlogConfig{ShowTags: false, HeadingSize: "large", ImageMode: "floating", ImageFit: "contain", ImageWidth: 78, ImageBackground: "#123456"}
	complete.Config.Deploy.Default = "preview"
	complete.Config.Deploy.Targets = map[string]config.DeployTarget{"preview": {Provider: "cloudflare-pages", AccountID: "account-123", Project: "test-project", ProductionBranch: "main", Domain: "docs.example.com"}}
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/config", complete, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("complete config update status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	full, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if full.Search.Enabled || full.Search.Shortcut != "Control+Shift+K" || full.Search.Placeholder != "Find the documentation" || full.Search.MaxResults != 18 || full.Search.RememberRecent || full.Accessibility.Shortcut != "Mod+A" || full.Theme.HoverColor != "#89a1ff" || full.Theme.HoverColorLight != "#223344" || full.Theme.HoverColorDark != "#ddeeff" || full.Theme.Layout.ContentWidth != "75%" || full.Theme.Layout.TOCWidth != "240px" || full.Social.GitHub != "https://example.com/project" || !full.Contribution.Enabled || full.Contribution.Repository != "https://github.com/example/docs.git" || full.Site.HeaderLinks[0].Label != "Home" || full.Deploy.Targets["preview"].Domain != "docs.example.com" || full.Blog.ShowTags || full.Blog.HeadingSize != "large" || full.Blog.ImageMode != "floating" || full.Blog.ImageFit != "contain" || full.Blog.ImageWidth != 78 || full.Blog.ImageBackground != "#123456" {
		t.Fatalf("complete configuration update was not applied: %#v", full)
	}
	css, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.css"))
	if err != nil || !strings.Contains(string(css), `--layout-content-width: clamp(30rem, 75%, 100rem);`) || !strings.Contains(string(css), `grid-template-columns: var(--layout-stage-fill) minmax(0, clamp(30rem, 75%, 100rem)) var(--layout-stage-fill) minmax(0, clamp(12rem, 240px, 25rem));`) {
		t.Fatalf("site was not rebuilt with the custom layout: %v\n%s", err, css)
	}
}

func TestDevelopmentConfigFormUsesMarkdownRenderer(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/__mpress/api/config-form")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("config form status %d: %s", response.StatusCode, readBody(response))
	}
	var payload struct {
		HTML string `json:"html"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<form class="mpress-form form-grid config-form"`,
		`id="config-form"`,
		`name="title"`,
		`name="colorScheme"`,
		`type="color"`,
	} {
		if !strings.Contains(payload.HTML, want) {
			t.Errorf("Markdown-rendered config form missing %q: %s", want, payload.HTML)
		}
	}
	if strings.Contains(payload.HTML, "@form") || strings.Contains(payload.HTML, "@end") {
		t.Fatalf("form directives leaked into config form: %s", payload.HTML)
	}
	if strings.Contains(payload.HTML, `type="reset"`) || strings.Contains(payload.HTML, `type="submit"`) {
		t.Fatalf("config form fragment must use the shared settings action bar: %s", payload.HTML)
	}

	siteResponse, err := http.Get(httpServer.URL + "/__mpress/api/config-form?section=site")
	if err != nil {
		t.Fatal(err)
	}
	defer siteResponse.Body.Close()
	if siteResponse.StatusCode != http.StatusOK {
		t.Fatalf("site config form status %d: %s", siteResponse.StatusCode, readBody(siteResponse))
	}
	var sitePayload struct {
		HTML string `json:"html"`
	}
	if err := json.NewDecoder(siteResponse.Body).Decode(&sitePayload); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="mpress-form-field"`, `name="site.title"`, `name="site.description"`, `name="site.baseURL"`, `name="accessibility.shortcut"`, `data-mpress-control="shortcut"`, `inputmode="none"`, `required`} {
		if !strings.Contains(sitePayload.HTML, want) {
			t.Errorf("Markdown-rendered site form missing %q: %s", want, sitePayload.HTML)
		}
	}
	if strings.Contains(sitePayload.HTML, "<form") || strings.Contains(sitePayload.HTML, "@form") {
		t.Fatalf("site settings form fragment leaked its wrapper or directives: %s", sitePayload.HTML)
	}
	if strings.Contains(sitePayload.HTML, `name="search.shortcut"`) {
		t.Fatalf("search shortcut is still shown under Site instead of Appearance: %s", sitePayload.HTML)
	}
	for _, section := range []string{"languages", "brand", "social", "build", "blog", "theme", "accessibility", "layout", "versioning", "translation", "deploy"} {
		sectionResponse, err := http.Get(httpServer.URL + "/__mpress/api/config-form?section=" + section)
		if err != nil {
			t.Fatal(err)
		}
		var sectionPayload struct {
			HTML string `json:"html"`
		}
		if sectionResponse.StatusCode != http.StatusOK {
			t.Fatalf("%s config form status %d: %s", section, sectionResponse.StatusCode, readBody(sectionResponse))
		}
		if err := json.NewDecoder(sectionResponse.Body).Decode(&sectionPayload); err != nil {
			sectionResponse.Body.Close()
			t.Fatal(err)
		}
		sectionResponse.Body.Close()
		if strings.Contains(sectionPayload.HTML, "<form") || strings.Contains(sectionPayload.HTML, "@form") || strings.TrimSpace(sectionPayload.HTML) == "" {
			t.Fatalf("%s settings form did not produce a usable Markdown fragment: %s", section, sectionPayload.HTML)
		}
		if section == "translation" && !strings.Contains(sectionPayload.HTML, `option value=""`) {
			t.Fatalf("translation settings form lost the provider-default empty option: %s", sectionPayload.HTML)
		}
		if section == "languages" && (!strings.Contains(sectionPayload.HTML, `name="site.defaultLanguage"`) || !strings.Contains(sectionPayload.HTML, `<select`)) {
			t.Fatalf("default language is not a select control: %s", sectionPayload.HTML)
		}
		if section == "brand" {
			for _, want := range []string{`type="file"`, `name="site.logoLightUpload"`, `name="site.logoDarkUpload"`, `name="site.faviconUpload"`, `name="site.socialImageUpload"`, `type="hidden"`, `name="site.logoLight"`, `name="site.logoDark"`, `name="site.favicon"`, `name="site.socialImage"`} {
				if !strings.Contains(sectionPayload.HTML, want) {
					t.Fatalf("brand settings form missing %q: %s", want, sectionPayload.HTML)
				}
			}
			if strings.Contains(sectionPayload.HTML, `name="site.logoWidth"`) {
				t.Fatalf("brand settings still expose manual logo sizing: %s", sectionPayload.HTML)
			}
		}
		if section == "social" {
			if want := `type="text" id="social-rss" name="social.rss" placeholder="/blog/rss.xml" autocomplete="url"`; !strings.Contains(sectionPayload.HTML, want) {
				t.Fatalf("social settings form does not permit a relative RSS URL, missing %q: %s", want, sectionPayload.HTML)
			}
			if strings.Contains(sectionPayload.HTML, `type="url" id="social-rss"`) {
				t.Fatalf("RSS field still requires an absolute URL: %s", sectionPayload.HTML)
			}
			for _, want := range []string{`Let readers contribute`, `Edit page URL base (optional)`, `placeholder="https://github.com/owner/docs/edit/main/content"`} {
				if !strings.Contains(sectionPayload.HTML, want) {
					t.Fatalf("social settings form is missing %q: %s", want, sectionPayload.HTML)
				}
			}
		}
		if section == "blog" {
			for _, want := range []string{`name="blog.landingStyle"`, `name="blog.tagline"`, `value="featured"`, `value="grid"`, `value="list"`} {
				if !strings.Contains(sectionPayload.HTML, want) {
					t.Fatalf("blog settings form missing %q: %s", want, sectionPayload.HTML)
				}
			}
		}
		if section == "theme" {
			for _, want := range []string{`<h2 id="colour-scheme">Colour Scheme</h2>`, `Default Scheme`, `<h2 id="search">Search</h2>`, `name="search.enabled"`, `name="search.rememberRecent"`, `name="search.placeholder"`, `name="search.maxResults"`, `name="search.shortcut"`} {
				if !strings.Contains(sectionPayload.HTML, want) {
					t.Fatalf("appearance settings form missing %q: %s", want, sectionPayload.HTML)
				}
			}
		}
	}
}

func TestBrandAssetUploadStoresThemeAssetInStaticBrandDirectory(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("slot", "logoLight"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("file", "brand-mark.svg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+"/__mpress/api/brand-asset", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("brand asset upload status %d: %s", response.StatusCode, readBody(response))
	}
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Path != "brand/logo-light.svg" {
		t.Fatalf("unexpected uploaded asset path %q", payload.Path)
	}
	if _, err := os.Stat(filepath.Join(root, "static", filepath.FromSlash(payload.Path))); err != nil {
		t.Fatalf("uploaded asset was not stored in static brand directory: %v", err)
	}
}

func TestEditedImageUploadRequiresOverwriteConfirmation(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52}
	upload := func(category, filename string, overwrite bool) *http.Response {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("category", category)
		_ = writer.WriteField("filename", filename)
		_ = writer.WriteField("overwrite", fmt.Sprint(overwrite))
		part, partErr := writer.CreateFormFile("file", filename)
		if partErr != nil {
			t.Fatal(partErr)
		}
		if _, partErr = part.Write(png); partErr != nil {
			t.Fatal(partErr)
		}
		if partErr = writer.Close(); partErr != nil {
			t.Fatal(partErr)
		}
		request, requestErr := http.NewRequest(http.MethodPost, httpServer.URL+"/__mpress/api/image-asset", &body)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		return response
	}

	response := upload("blog", "release-cover.png", false)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("first image save returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	if _, err := os.Stat(filepath.Join(root, "static", "images", "blog", "release-cover.png")); err != nil {
		t.Fatalf("edited image was not stored: %v", err)
	}
	response = upload("blog", "release-cover.png", false)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("unconfirmed overwrite returned %d: %s", response.StatusCode, readBody(response))
	}
	var conflict map[string]string
	if err := json.NewDecoder(response.Body).Decode(&conflict); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if conflict["suggestedName"] != "release-cover-edited.png" {
		t.Fatalf("unexpected alternative filename: %#v", conflict)
	}
	response = upload("blog", "release-cover.png", true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("confirmed overwrite returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	response = upload("content", "architecture.png", false)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("content image save returned %d: %s", response.StatusCode, readBody(response))
	}
	var saved map[string]string
	if err := json.NewDecoder(response.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if saved["path"] != "images/architecture.png" {
		t.Fatalf("unexpected content image path: %#v", saved)
	}
	if _, err := os.Stat(filepath.Join(root, "static", "images", "architecture.png")); err != nil {
		t.Fatalf("content image was not stored in the shared images directory: %v", err)
	}
}

func TestDevelopmentTranslationWizardAddsLanguageAndRebuilds(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/translations", map[string]any{
		"action": "add-language", "language": "fr", "label": "Français",
	}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("add language returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Site.Languages) != 2 || cfg.Site.Languages[1] != "fr" || cfg.Site.LanguageLabels["fr"] != "Français" {
		t.Fatalf("translation wizard did not persist the new language: %#v", cfg.Site)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil || !strings.Contains(string(page), "Français") {
		t.Fatalf("rebuilt language selector does not include the new language: %v\n%s", err, page)
	}
}

func TestDevelopmentTranslationWizardConvertsMarkdownToNativeMPD(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	get, err := http.Get(httpServer.URL + "/__mpress/api/translations")
	if err != nil {
		t.Fatal(err)
	}
	var before struct {
		DocumentFormats struct {
			Markdown int  `json:"markdown"`
			MPD      int  `json:"mpd"`
			Native   bool `json:"nativeMPD"`
		} `json:"documentFormats"`
	}
	if err := json.NewDecoder(get.Body).Decode(&before); err != nil {
		t.Fatal(err)
	}
	get.Body.Close()
	if before.DocumentFormats.Markdown == 0 || before.DocumentFormats.Native {
		t.Fatalf("unexpected initial format summary: %#v", before.DocumentFormats)
	}

	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/translations", map[string]any{"action": "convert-mpd"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("convert to MPD returned %d: %s", response.StatusCode, readBody(response))
	}
	var converted struct {
		Conversion struct {
			Count        int    `json:"count"`
			SourceFormat string `json:"sourceFormat"`
			TargetFormat string `json:"targetFormat"`
		} `json:"conversion"`
	}
	if err := json.NewDecoder(response.Body).Decode(&converted); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if converted.Conversion.Count == 0 || converted.Conversion.SourceFormat != "Markdown" || converted.Conversion.TargetFormat != "MPD" {
		t.Fatalf("unexpected conversion result: %#v", converted.Conversion)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "content", "*.md")); len(matches) != 0 {
		t.Fatalf("Markdown files remain after conversion: %v", matches)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "content", "*.mpd")); len(matches) == 0 {
		t.Fatal("native MPD files were not created")
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil || !strings.Contains(string(page), "Editable heading") {
		t.Fatalf("strictly rebuilt MPD output is missing: %v\n%s", err, page)
	}
}

func TestDevelopmentRepositorySetupCreatesWorkBranch(t *testing.T) {
	root := authoringFixture(t)
	for _, args := range [][]string{
		{"init", "-b", "main", root},
		{"-C", root, "config", "user.email", "mpress@example.test"},
		{"-C", root, "config", "user.name", "M-Press Test"},
		{"-C", root, "add", "mpress.yaml", "content"},
		{"-C", root, "commit", "-m", "Initial documentation"},
	} {
		command := exec.Command("git", args...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/setup/repository", repositorySetupPayload{Action: "current", Branch: "docs/mpress-mvp"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("repository setup returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	if branch := strings.TrimSpace(gitOutput(root, "branch", "--show-current")); branch != "docs/mpress-mvp" {
		t.Fatalf("repository setup left branch %q", branch)
	}
	info := inspectRepository(root)
	if !info.IsGit || !info.OnWorkBranch {
		t.Fatalf("prepared repository is not reported as ready: %#v", info)
	}
	if err := createWorkBranch(context.Background(), root, "main"); err == nil || !strings.Contains(err.Error(), "default branch") {
		t.Fatalf("repository setup allowed editing on the default branch: %v", err)
	}

	cloneDirectory := filepath.Join(t.TempDir(), "checkout")
	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/setup/repository", repositorySetupPayload{
		Action: "clone", Repository: root, Destination: cloneDirectory, Branch: "docs/cloned-mpress-mvp",
	}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("clone setup returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	if branch := strings.TrimSpace(gitOutput(cloneDirectory, "branch", "--show-current")); branch != "docs/cloned-mpress-mvp" {
		t.Fatalf("cloned repository setup left branch %q", branch)
	}
	if _, err := os.Stat(filepath.Join(cloneDirectory, config.Filename)); err != nil {
		t.Fatalf("cloned repository is missing %s: %v", config.Filename, err)
	}
}

func TestDevelopmentDeploymentSetupAndOnboardingArePersistedSafely(t *testing.T) {
	root := authoringFixture(t)
	if err := os.MkdirAll(filepath.Join(root, ".mpress"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".mpress", "onboarding"), []byte("start"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, _ := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	setup := deploymentPayload{AccountID: "account-123", Project: "test-docs", ProductionBranch: "main"}
	response := requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/deployment", setup, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("deployment setup status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Deploy.Default != "cloudflare" || cfg.Deploy.Targets["cloudflare"].Project != "test-docs" {
		t.Fatalf("deployment target was not saved: %#v", cfg.Deploy)
	}
	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/onboarding", map[string]any{}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("onboarding status %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	if _, err := os.Stat(filepath.Join(root, ".mpress", "onboarding")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("onboarding marker still exists: %v", err)
	}
}

func TestFirstRunOnboardingCanReplaceTheTutorialWithAMinimalSite(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "content/getting-started.md", "# Tutorial\n")
	writeDevFixture(t, root, "content/components.md", "# Components\n")
	writeDevFixture(t, root, "static/images/component-light.svg", "<svg/>")
	writeDevFixture(t, root, ".mpress/onboarding", "start\n")
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/onboarding", map[string]string{"action": "starter", "starter": "empty"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("empty starter returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	for _, name := range []string{"content/getting-started.md", "content/components.md", "static/images/component-light.svg"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("starter file %s still exists: %v", name, err)
		}
	}
	index, err := os.ReadFile(filepath.Join(root, "content", "index.md"))
	if err != nil || !strings.Contains(string(index), "title: Welcome") || strings.Contains(string(index), "# Welcome") {
		t.Fatalf("minimal home page was not written: %v\n%s", err, index)
	}
	if !server.onboarding() {
		t.Fatal("choosing starter content completed onboarding too early")
	}
}

func TestContributionCheckoutCanReviewCommitAndPushChanges(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "CONTRIBUTING.md", "# Contribution guide\n\nKeep changes focused.\n")
	mustGit(t, root, "init", "-b", "main")
	mustGit(t, root, "config", "user.name", "MPress Test")
	mustGit(t, root, "config", "user.email", "mpress@example.test")
	mustGit(t, root, "add", "--all")
	mustGit(t, root, "commit", "-m", "initial")
	start := strings.TrimSpace(mustGit(t, root, "rev-parse", "HEAD"))
	origin := filepath.Join(t.TempDir(), "origin.git")
	if output, err := exec.Command("git", "init", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("init bare origin: %v\n%s", err, output)
	}
	mustGit(t, root, "remote", "add", "origin", origin)
	mustGit(t, root, "push", "--set-upstream", "origin", "main")
	mustGit(t, root, "checkout", "-b", "contribute/test")
	writeDevFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Improved heading\n")

	session := &ContributionSession{Repository: origin, SourcePath: "content/index.md", Route: "/", Branch: "contribute/test", StartCommit: start}
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true, Contribution: session})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	projectResponse, err := http.Get(httpServer.URL + "/__mpress/api/project")
	if err != nil {
		t.Fatal(err)
	}
	var project struct {
		ContributorGuide contributorGuide `json:"contributorGuide"`
	}
	if err := json.NewDecoder(projectResponse.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}
	projectResponse.Body.Close()
	if project.ContributorGuide.Path != "CONTRIBUTING.md" || !strings.Contains(project.ContributorGuide.Markdown, "Keep changes focused") {
		t.Fatalf("contributor guide missing from project: %#v", project.ContributorGuide)
	}

	statusResponse, err := http.Get(httpServer.URL + "/__mpress/api/contribution")
	if err != nil {
		t.Fatal(err)
	}
	var before contributionStatus
	if err := json.NewDecoder(statusResponse.Body).Decode(&before); err != nil {
		t.Fatal(err)
	}
	statusResponse.Body.Close()
	if before.Clean || len(before.Files) != 1 || !strings.Contains(before.Diff, "Improved heading") {
		t.Fatalf("unexpected contribution review: %#v", before)
	}

	commitResponse := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/contribution", contributionAction{Action: "commit", Message: "docs: improve heading"}, "")
	if commitResponse.StatusCode != http.StatusOK {
		t.Fatalf("commit returned %d: %s", commitResponse.StatusCode, readBody(commitResponse))
	}
	var committed contributionStatus
	if err := json.NewDecoder(commitResponse.Body).Decode(&committed); err != nil {
		t.Fatal(err)
	}
	commitResponse.Body.Close()
	if !committed.Clean || !committed.Committed || committed.Ahead != 1 {
		t.Fatalf("contribution was not committed: %#v", committed)
	}

	pushResponse := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/contribution", contributionAction{Action: "push"}, "")
	if pushResponse.StatusCode != http.StatusOK {
		t.Fatalf("push returned %d: %s", pushResponse.StatusCode, readBody(pushResponse))
	}
	var pushed contributionStatus
	if err := json.NewDecoder(pushResponse.Body).Decode(&pushed); err != nil {
		t.Fatal(err)
	}
	pushResponse.Body.Close()
	if !pushed.Pushed || pushed.Branch != "contribute/test" {
		t.Fatalf("contribution branch was not pushed: %#v", pushed)
	}
}

func mustGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", directory}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func TestDevelopmentBlogEditorCreatesAndUpdatesMarkdown(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "content/blog/index.md", "---\ntitle: Blog\n---\n\n# Blog\n")
	writeDevFixture(t, root, "content/blog/existing.md", "---\ntitle: Existing post\ndate: 2026-08-01\ncustom: preserve-me\n---\n\nOriginal body.\n")
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/__mpress/api/blog-posts")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Posts []blogPostResponse `json:"posts"`
	}
	if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()
	if len(list.Posts) != 1 || list.Posts[0].Date != "2026-08-01" || list.Posts[0].Path != "content/blog/existing.md" {
		t.Fatalf("unexpected blog post list: %#v", list.Posts)
	}

	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/blog-posts", blogPostPayload{
		Title: "A new post", Slug: "a-new-post", Date: "2026-08-08", Description: "A concise summary.",
		Author: "M-Press", Tags: []string{"release", "docs"}, Draft: true, Body: "## What changed\n\nThe editor writes **Markdown**.",
	}, "")
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create blog post returned %d: %s", response.StatusCode, readBody(response))
	}
	var created struct {
		Post blogPostResponse `json:"post"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()
	createdData, err := os.ReadFile(filepath.Join(root, "content", "blog", "a-new-post.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"title: A new post", "description: A concise summary.", "draft: true", "## What changed", "**Markdown**"} {
		if !strings.Contains(string(createdData), want) {
			t.Errorf("created Markdown post missing %q: %s", want, createdData)
		}
	}

	detail, err := http.Get(httpServer.URL + "/__mpress/api/blog-posts?path=content/blog/existing.md")
	if err != nil {
		t.Fatal(err)
	}
	var existing blogPostResponse
	if err := json.NewDecoder(detail.Body).Decode(&existing); err != nil {
		detail.Body.Close()
		t.Fatal(err)
	}
	detail.Body.Close()
	response = requestJSON(t, http.MethodPut, httpServer.URL+"/__mpress/api/blog-posts", blogPostPayload{
		Path: existing.Path, Revision: existing.Revision, Title: "Updated post", Date: "2026-08-02", Body: "Updated body.",
	}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("update blog post returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	updated, err := os.ReadFile(filepath.Join(root, "content", "blog", "existing.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "custom: preserve-me") || !strings.Contains(string(updated), "title: Updated post") || !strings.Contains(string(updated), "Updated body.") {
		t.Fatalf("updated Markdown post lost content or custom frontmatter: %s", updated)
	}

	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Bad slug", Slug: "Not Valid"}, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid blog slug returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/markdown-preview", map[string]string{"markdown": "## Preview\n\nRendered **safely**."}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Markdown preview returned %d: %s", response.StatusCode, readBody(response))
	}
	var preview struct {
		HTML string `json:"html"`
	}
	if err := json.NewDecoder(response.Body).Decode(&preview); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()
	if !strings.Contains(preview.HTML, "<h2") || !strings.Contains(preview.HTML, "<strong>safely</strong>") {
		t.Fatalf("Markdown preview did not use the M-Press renderer: %s", preview.HTML)
	}
}

func TestDevelopmentVersionManagerCapturesVerifiesSelectsAndRemoves(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.rebuild(false)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	for _, label := range []string{"3.0.0", "2.0"} {
		response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/versions", versionActionPayload{Action: "capture", Label: label}, "")
		if response.StatusCode != http.StatusOK {
			t.Fatalf("capture %s returned %d: %s", label, response.StatusCode, readBody(response))
		}
		response.Body.Close()
	}
	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/versions", versionActionPayload{Action: "verify", Label: "3.0.0"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("verify returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/versions", versionActionPayload{Action: "current", Label: "2.0"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("set current returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Version.Enabled || cfg.Version.Current != "2.0" {
		t.Fatalf("version selection was not persisted: %#v", cfg.Version)
	}
	response = requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/versions", versionActionPayload{Action: "remove", Label: "3.0.0"}, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("remove returned %d: %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	if _, err := os.Stat(filepath.Join(cfg.ArtifactsPath(root), "3.0.0")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed version still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.ArtifactsPath(root), "2.0", "mpress-version.json")); err != nil {
		t.Fatalf("current version artifact is missing: %v", err)
	}
}

func authoringFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeDevFixture(t, root, "mpress.yaml", "site:\n  title: Test project\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\n")
	writeDevFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Editable heading\n\nOriginal **paragraph** with emphasis.\n")
	writeDevFixture(t, root, "content/_nav.yaml", "- label: Home\n  link: /\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeDevFixture(t *testing.T, root, name, value string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func getFile(t *testing.T, baseURL, path string) filePayload {
	t.Helper()
	response, err := http.Get(baseURL + "/__mpress/api/file?path=" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload filePayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func requestJSON(t *testing.T, method, url string, payload any, token string) *http.Response {
	t.Helper()
	data, _ := json.Marshal(payload)
	request, _ := http.NewRequest(method, url, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("X-MPress-Token", token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func readBody(response *http.Response) string {
	data, _ := io.ReadAll(response.Body)
	return string(data)
}
