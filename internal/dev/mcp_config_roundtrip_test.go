package dev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// configCollectionPaths are the optional collection fields of the complete
// configuration. Their JSON Schema types are object and array, so a nil Go
// value must be represented as an empty collection rather than null.
var configCollectionPaths = []struct {
	path     string
	kind     string
	optional bool
}{
	{"site.languages", "array", false},
	{"site.languageLabels", "object", false},
	{"site.headerLinks", "array", false},
	// languageModels is optional in the declared configuration schema.
	{"translation.languageModels", "object", true},
	{"deploy.targets", "object", false},
}

func newMCPConfigTestServer(t *testing.T, root string, authoring bool, token string) (*httptest.Server, *mcp.ClientSession) {
	t.Helper()
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: authoring, Token: token})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.rebuild(false)
	host := httptest.NewServer(server.Handler())
	t.Cleanup(host.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "mpress-config-test", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             host.URL + "/__mpress/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: host.Client().Transport, token: token}},
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return host, session
}

func callMCPTool(t *testing.T, session *mcp.ClientSession, name string, arguments any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("%s transport error: %v", name, err)
	}
	return result
}

func mcpToolError(result *mcp.CallToolResult) string {
	if !result.IsError {
		return ""
	}
	var messages []string
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			messages = append(messages, text.Text)
		}
	}
	return strings.Join(messages, "\n")
}

func requireMCPToolSuccess(t *testing.T, name string, result *mcp.CallToolResult) {
	t.Helper()
	if result.IsError {
		t.Fatalf("%s failed: %s", name, mcpToolError(result))
	}
}

func mcpConfigPayload(t *testing.T, session *mcp.ClientSession) (map[string]any, string) {
	t.Helper()
	result := callMCPTool(t, session, "get_config", nil)
	requireMCPToolSuccess(t, "get_config", result)
	payload := structuredMCPOutput(t, result)
	complete, ok := payload["config"].(map[string]any)
	if !ok {
		t.Fatalf("get_config output has no configuration object: %#v", payload)
	}
	revision, ok := payload["revision"].(string)
	if !ok || revision == "" {
		t.Fatalf("get_config output has no revision: %#v", payload)
	}
	return complete, revision
}

func updateMCPConfig(t *testing.T, session *mcp.ClientSession, complete map[string]any, revision string) map[string]any {
	t.Helper()
	result := callMCPTool(t, session, "update_config", map[string]any{"config": complete, "revision": revision})
	requireMCPToolSuccess(t, "update_config", result)
	return structuredMCPOutput(t, result)
}

func rejectMCPTool(t *testing.T, session *mcp.ClientSession, name string, arguments any) {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return
	}
	if !result.IsError {
		t.Fatalf("%s unexpectedly succeeded: %#v", name, result.StructuredContent)
	}
}

func structuredMCPOutput(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured output: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode structured output: %v", err)
	}
	return payload
}

func configJSONValue(t *testing.T, complete map[string]any, path string) any {
	t.Helper()
	value, ok := configJSONValueIfPresent(complete, path)
	if !ok {
		t.Fatalf("configuration path %q is missing", path)
	}
	return value
}

func configJSONValueIfPresent(complete map[string]any, path string) (any, bool) {
	var current any = complete
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok := object[part]
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func requireCollectionKind(t *testing.T, complete map[string]any, path, kind string, optional bool) {
	t.Helper()
	value, present := configJSONValueIfPresent(complete, path)
	if !present {
		if optional {
			return
		}
		t.Errorf("%s is missing, want JSON %s", path, kind)
		return
	}
	switch kind {
	case "object":
		if object, ok := value.(map[string]any); !ok || object == nil {
			t.Errorf("%s = %#v, want JSON object (not null)", path, value)
		}
	case "array":
		if array, ok := value.([]any); !ok || array == nil {
			t.Errorf("%s = %#v, want JSON array (not null)", path, value)
		}
	default:
		t.Fatalf("unknown collection kind %q", kind)
	}
}

func requireConfigCollectionShape(t *testing.T, complete map[string]any) {
	t.Helper()
	for _, collection := range configCollectionPaths {
		requireCollectionKind(t, complete, collection.path, collection.kind, collection.optional)
	}
}

func restConfigDocument(t *testing.T, host *httptest.Server) (map[string]any, map[string]any) {
	t.Helper()
	payload := getConfigPayload(t, host.URL)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	complete, ok := document["config"].(map[string]any)
	if !ok {
		t.Fatalf("REST config has no complete configuration: %#v", document)
	}
	return document, complete
}

func requireRenderedTitle(t *testing.T, host *httptest.Server, title string) {
	t.Helper()
	response, err := http.Get(host.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(response)
	_ = response.Body.Close()
	if !strings.Contains(body, title) {
		t.Fatalf("rendered site does not contain title %q", title)
	}
}

func TestMCPConfigRoundTripDefaultOptionalCollections(t *testing.T) {
	root := authoringFixture(t)
	host, session := newMCPConfigTestServer(t, root, true, "config-roundtrip")

	complete, sourceRevision := mcpConfigPayload(t, session)
	requireConfigCollectionShape(t, complete)

	// The fixture omits every optional collection. Non-omitempty fields must
	// arrive as empty objects and arrays so a client can send the document back
	// unchanged; omitempty fields may be absent but must never be null.
	for _, collection := range configCollectionPaths {
		value, present := configJSONValueIfPresent(complete, collection.path)
		if !present {
			continue
		}
		switch value := value.(type) {
		case map[string]any:
			if collection.path == "site.languageLabels" && len(value) != 0 {
				t.Errorf("unexpected languageLabels %#v", value)
			}
		case []any:
			if collection.path == "site.headerLinks" && len(value) != 0 {
				t.Errorf("unexpected headerLinks %#v", value)
			}
		}
	}

	const title = "MCP roundtrip default"
	complete["site"].(map[string]any)["title"] = title
	update := updateMCPConfig(t, session, complete, sourceRevision)
	if update["revision"] == "" || update["revision"] == nil {
		t.Fatalf("update_config returned no revision: %#v", update)
	}

	after, afterRevision := mcpConfigPayload(t, session)
	if got := configJSONValue(t, after, "site.title"); got != title {
		t.Fatalf("get_config title after update = %#v, want %q", got, title)
	}
	if afterRevision != update["revision"] {
		t.Fatalf("get_config revision %q != update_config revision %q", afterRevision, update["revision"])
	}
	requireConfigCollectionShape(t, after)

	persisted, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Site.Title != title {
		t.Fatalf("persisted title = %q, want %q", persisted.Site.Title, title)
	}

	requireRenderedTitle(t, host, title)

	restPayload, restComplete := restConfigDocument(t, host)
	if restPayload["revision"] != update["revision"] || configJSONValue(t, restComplete, "site.title") != title {
		t.Fatalf("REST config did not converge: %#v", restPayload)
	}
	requireConfigCollectionShape(t, restComplete)
}

func TestMCPConfigRoundTripPopulatedOptionalCollections(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "content/docs.md", "# Docs\n")
	writeDevFixture(t, root, "content/support.md", "# Support\n")
	writeDevFixture(t, root, config.Filename, `site:
  title: Test project
  defaultLanguage: en
  languages: [en, fr]
  languageLabels:
    en: English
    fr: Français
  headerLinks:
    - label: Docs
      url: /docs/
    - label: Support
      url: /support/
      type: button
      variant: primary
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
translation:
  provider: openrouter
  languageModels:
    fr:
      model: example/translation-model
      reasoningEffort: low
deploy:
  default: preview
  targets:
    preview:
      provider: netlify
      project: docs-preview
`)

	host, session := newMCPConfigTestServer(t, root, true, "config-populated")
	complete, sourceRevision := mcpConfigPayload(t, session)
	requireConfigCollectionShape(t, complete)

	if got := configJSONValue(t, complete, "site.languageLabels"); len(got.(map[string]any)) != 2 || got.(map[string]any)["fr"] != "Français" {
		t.Fatalf("languageLabels were not preserved: %#v", got)
	}
	links, ok := configJSONValue(t, complete, "site.headerLinks").([]any)
	if !ok || len(links) != 2 {
		t.Fatalf("headerLinks were not preserved: %#v", links)
	}
	if link, ok := links[1].(map[string]any); !ok || link["label"] != "Support" || link["variant"] != "primary" {
		t.Fatalf("header link values were not preserved: %#v", links[1])
	}
	models := configJSONValue(t, complete, "translation.languageModels").(map[string]any)
	if selection, ok := models["fr"].(map[string]any); !ok || selection["model"] != "example/translation-model" || selection["reasoningEffort"] != "low" {
		t.Fatalf("languageModels were not preserved: %#v", models)
	}
	targets := configJSONValue(t, complete, "deploy.targets").(map[string]any)
	if target, ok := targets["preview"].(map[string]any); !ok || target["project"] != "docs-preview" {
		t.Fatalf("deploy targets were not preserved: %#v", targets)
	}

	const title = "MCP roundtrip populated"
	complete["site"].(map[string]any)["title"] = title
	update := updateMCPConfig(t, session, complete, sourceRevision)

	after, afterRevision := mcpConfigPayload(t, session)
	if configJSONValue(t, after, "site.title") != title {
		t.Fatalf("title after update = %#v", configJSONValue(t, after, "site.title"))
	}
	if afterRevision != update["revision"] {
		t.Fatalf("get_config revision %q != update_config revision %q", afterRevision, update["revision"])
	}
	requireConfigCollectionShape(t, after)
	if configJSONValue(t, after, "site.languageLabels").(map[string]any)["fr"] != "Français" {
		t.Fatalf("languageLabels changed after update: %#v", configJSONValue(t, after, "site.languageLabels"))
	}
	if link := configJSONValue(t, after, "site.headerLinks").([]any)[1].(map[string]any); link["label"] != "Support" {
		t.Fatalf("headerLinks changed after update: %#v", link)
	}
	if selection := configJSONValue(t, after, "translation.languageModels").(map[string]any)["fr"].(map[string]any); selection["reasoningEffort"] != "low" {
		t.Fatalf("languageModels changed after update: %#v", selection)
	}
	if target := configJSONValue(t, after, "deploy.targets").(map[string]any)["preview"].(map[string]any); target["project"] != "docs-preview" {
		t.Fatalf("deploy targets changed after update: %#v", target)
	}

	persisted, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Site.Title != title || persisted.Site.LanguageLabels["fr"] != "Français" || len(persisted.Site.HeaderLinks) != 2 || persisted.Site.HeaderLinks[1].Variant != "primary" || persisted.Translation.LanguageModels["fr"].Model != "example/translation-model" || persisted.Deploy.Targets["preview"].Project != "docs-preview" {
		t.Fatalf("persisted configuration lost populated values: %#v", persisted)
	}

	requireRenderedTitle(t, host, title)

	restPayload, restComplete := restConfigDocument(t, host)
	if restPayload["revision"] != update["revision"] || configJSONValue(t, restComplete, "site.title") != title {
		t.Fatalf("REST config did not converge: %#v", restPayload)
	}
	requireConfigCollectionShape(t, restComplete)
	if configJSONValue(t, restComplete, "site.languageLabels").(map[string]any)["fr"] != "Français" {
		t.Fatalf("REST config lost languageLabels: %#v", restComplete)
	}
	if configJSONValue(t, restComplete, "site.headerLinks").([]any)[0].(map[string]any)["label"] != "Docs" {
		t.Fatalf("REST config lost headerLinks: %#v", restComplete)
	}
}

func TestMCPConfigRejectsMalformedCollectionsAndRevisions(t *testing.T) {
	root := authoringFixture(t)
	path := filepath.Join(root, config.Filename)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, session := newMCPConfigTestServer(t, root, true, "config-validation")

	cases := []struct {
		name   string
		mutate func(complete map[string]any)
	}{
		{"languageLabels scalar", func(c map[string]any) { c["site"].(map[string]any)["languageLabels"] = "English" }},
		{"languageLabels element", func(c map[string]any) {
			c["site"].(map[string]any)["languageLabels"] = map[string]any{"en": 42}
		}},
		{"languages scalar", func(c map[string]any) { c["site"].(map[string]any)["languages"] = "/docs" }},
		{"languages element", func(c map[string]any) { c["site"].(map[string]any)["languages"] = []any{"en", 42} }},
		{"headerLinks object", func(c map[string]any) {
			c["site"].(map[string]any)["headerLinks"] = map[string]any{"label": "Home"}
		}},
		{"headerLinks element", func(c map[string]any) { c["site"].(map[string]any)["headerLinks"] = []any{"Home"} }},
		{"headerLinks missing url", func(c map[string]any) {
			c["site"].(map[string]any)["headerLinks"] = []any{map[string]any{"label": "Home"}}
		}},
		{"languageModels element", func(c map[string]any) {
			c["translation"].(map[string]any)["languageModels"] = map[string]any{"fr": "example/model"}
		}},
		{"deploy targets scalar", func(c map[string]any) { c["deploy"].(map[string]any)["targets"] = "preview" }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			complete, revision := mcpConfigPayload(t, session)
			testCase.mutate(complete)
			rejectMCPTool(t, session, "update_config", map[string]any{"config": complete, "revision": revision})
			after, readErr := os.ReadFile(path)
			if readErr != nil || string(after) != string(original) {
				t.Fatalf("rejected configuration changed the source: %v", readErr)
			}
		})
	}

	t.Run("missing revision", func(t *testing.T) {
		complete, _ := mcpConfigPayload(t, session)
		rejectMCPTool(t, session, "update_config", map[string]any{"config": complete})
		after, readErr := os.ReadFile(path)
		if readErr != nil || string(after) != string(original) {
			t.Fatalf("missing revision changed the source: %v", readErr)
		}
	})

	t.Run("empty revision", func(t *testing.T) {
		complete, _ := mcpConfigPayload(t, session)
		rejectMCPTool(t, session, "update_config", map[string]any{"config": complete, "revision": ""})
		after, readErr := os.ReadFile(path)
		if readErr != nil || string(after) != string(original) {
			t.Fatalf("empty revision changed the source: %v", readErr)
		}
	})

	t.Run("stale revision", func(t *testing.T) {
		complete, _ := mcpConfigPayload(t, session)
		rejectMCPTool(t, session, "update_config", map[string]any{"config": complete, "revision": "stale-revision"})
		after, readErr := os.ReadFile(path)
		if readErr != nil || string(after) != string(original) {
			t.Fatalf("stale revision changed the source: %v", readErr)
		}
	})
}

func TestMCPConfigHonoursReadOnlyBearerAndExternalConfigBounds(t *testing.T) {
	t.Run("read only", func(t *testing.T) {
		root := authoringFixture(t)
		path := filepath.Join(root, config.Filename)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, session := newMCPConfigTestServer(t, root, false, "config-read-only")

		complete, revision := mcpConfigPayload(t, session)
		requireConfigCollectionShape(t, complete)
		complete["site"].(map[string]any)["title"] = "Read-only edit"
		rejectMCPTool(t, session, "update_config", map[string]any{"config": complete, "revision": revision})
		after, readErr := os.ReadFile(path)
		if readErr != nil || string(after) != string(original) {
			t.Fatalf("read-only update changed the source: %v", readErr)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		root := authoringFixture(t)
		path := filepath.Join(root, config.Filename)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true, Token: "config-bearer"})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = server.Close() })
		host := httptest.NewServer(server.Handler())
		t.Cleanup(host.Close)

		response, err := http.Post(host.URL+"/__mpress/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"update_config","arguments":{}}}`))
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("unauthenticated MCP call returned %d", response.StatusCode)
		}
		after, readErr := os.ReadFile(path)
		if readErr != nil || string(after) != string(original) {
			t.Fatalf("unauthenticated call changed the source: %v", readErr)
		}
	})

	t.Run("external configuration alias", func(t *testing.T) {
		root, outside := authoringFixture(t), t.TempDir()
		path := filepath.Join(root, config.Filename)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, session := newMCPConfigTestServer(t, root, true, "config-alias")

		writeDevFixture(t, outside, config.Filename, string(original))
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		devSymlink(t, filepath.Join(outside, config.Filename), path)

		rejectMCPTool(t, session, "get_config", nil)
		external, err := os.ReadFile(filepath.Join(outside, config.Filename))
		if err != nil || string(external) != string(original) {
			t.Fatalf("external configuration alias changed: %v", err)
		}
	})
}
