package knowledge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestKnowledgeMCPIsReadOnlySearchableAndCitable(t *testing.T) {
	page := Page{PageSummary: PageSummary{ID: "page-1", ResourceURI: "mpress://knowledge/page/page-1", Language: "en", Version: "current", Route: "/install/", URL: "https://docs.example.test/install/", Source: "install.md", Title: "Install", Description: "Install the product.", Tags: []string{"setup"}, ChunkCount: 1}, Text: "Install the Linux package and configure a token."}
	chunk := Chunk{ID: "chunk-1", PageID: page.ID, ResourceURI: "mpress://knowledge/section/chunk-1", Language: "en", Version: "current", Route: page.Route, URL: page.URL + "#linux", Source: page.Source, PageTitle: page.Title, Title: "Linux", HeadingID: "linux", HeadingPath: []string{"Install", "Linux"}, Tags: page.Tags, Text: "Install the Linux package."}
	store := newStore(Manifest{Schema: Schema, Title: "Example", DefaultLanguage: "en", Languages: []string{"en"}, CurrentVersion: "current", Digest: "digest"}, []Page{page}, []Chunk{chunk}, makeIndex([]Chunk{chunk}))
	server := NewMCPServer(store, "test")
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "knowledge-test", Version: "test"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	resources, err := clientSession.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources.Resources) != 1 || resources.Resources[0].URI != page.ResourceURI {
		t.Fatalf("unexpected resources: %#v", resources.Resources)
	}
	read, err := clientSession.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: page.ResourceURI})
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Contents) != 1 || read.Contents[0].Text == "" || read.CacheScope != "public" || read.TTLMs == 0 {
		t.Fatalf("unexpected page resource: %#v", read)
	}
	search, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: "search", Arguments: map[string]any{"query": "Linux package"}})
	if err != nil {
		t.Fatal(err)
	}
	if search.IsError || len(search.Content) != 1 {
		t.Fatalf("unexpected search result: %#v", search)
	}
	link, ok := search.Content[0].(*mcp.ResourceLink)
	if !ok || link.URI != chunk.ResourceURI {
		t.Fatalf("search did not return the section resource: %#v", search.Content)
	}
	section, err := clientSession.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: chunk.ResourceURI})
	if err != nil {
		t.Fatal(err)
	}
	if len(section.Contents) != 1 || section.Contents[0].Text != "Linux\n\nCitation: https://docs.example.test/install/#linux\n\nInstall the Linux package." {
		t.Fatalf("unexpected section resource: %#v", section.Contents)
	}
	tools, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 3 {
		t.Fatalf("tools=%d, want exactly three read-only tools", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
			t.Fatalf("tool %q is not declared read-only: %#v", tool.Name, tool.Annotations)
		}
	}
}

func TestKnowledgeMCPStreamableHTTPTransport(t *testing.T) {
	page := Page{PageSummary: PageSummary{ID: "page-1", ResourceURI: "mpress://knowledge/page/page-1", Language: "en", Version: "current", Route: "/", URL: "https://docs.example.test/", Source: "index.md", Title: "Home", ChunkCount: 1}, Text: "Welcome."}
	chunk := Chunk{ID: "chunk-1", PageID: page.ID, ResourceURI: "mpress://knowledge/section/chunk-1", Language: "en", Version: "current", Route: "/", URL: page.URL, Source: page.Source, PageTitle: page.Title, Title: "Home", Text: "Welcome."}
	store := newStore(Manifest{Schema: Schema, Title: "Example", DefaultLanguage: "en", Languages: []string{"en"}, CurrentVersion: "current", Digest: "digest"}, []Page{page}, []Chunk{chunk}, makeIndex([]Chunk{chunk}))
	mux := http.NewServeMux()
	mux.Handle("/mcp", HTTPHandler(store, "test"))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "knowledge-http-test", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "site_info", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || result.StructuredContent == nil {
		t.Fatalf("unexpected site_info response: %#v", result)
	}
}
