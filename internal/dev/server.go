package dev

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/exportzip"
	"github.com/leaanthony/mpress/internal/knowledge"
	"github.com/leaanthony/mpress/internal/lighthouse"
	"github.com/leaanthony/mpress/internal/operations"
	"github.com/leaanthony/mpress/internal/projectconvert"
	"github.com/leaanthony/mpress/internal/site"
	"github.com/leaanthony/mpress/internal/translate"
	docversion "github.com/leaanthony/mpress/internal/version"
	"gopkg.in/yaml.v3"
)

const maxSourceSize = 5 << 20
const maxBrandAssetSize = 2 << 20

type Options struct {
	Host         string
	Port         int
	AutoPort     bool
	OpenBrowser  bool
	Authoring    bool
	Token        string
	Version      string
	Workspace    *WorkspaceAssets
	Contribution *ContributionSession
}

// ContributionSession describes the public page that launched a local
// contributor checkout. It contains no credentials.
type ContributionSession struct {
	SiteURL     string `json:"siteURL"`
	Repository  string `json:"repository"`
	SourcePath  string `json:"sourcePath,omitempty"`
	Route       string `json:"route,omitempty"`
	Branch      string `json:"branch"`
	Guide       string `json:"guide,omitempty"`
	Goal        string `json:"goal,omitempty"`
	StartCommit string `json:"startCommit,omitempty"`
}

type WorkspaceAssets struct {
	HTML string
	CSS  string
	JS   string
}

type BuildState struct {
	Status      string               `json:"status"`
	Pages       int                  `json:"pages"`
	Files       int                  `json:"files"`
	DurationMS  int64                `json:"durationMs"`
	Diagnostics []content.Diagnostic `json:"diagnostics,omitempty"`
	Error       string               `json:"error,omitempty"`
	BuiltAt     time.Time            `json:"builtAt"`
	Revision    int64                `json:"revision"`
}

type Server struct {
	project                string
	cfg                    config.Config
	options                Options
	token                  string
	requireWriteToken      bool
	events                 *hub
	stateMu                sync.RWMutex
	state                  BuildState
	buildMu                sync.Mutex
	previewMu              sync.Mutex
	previewDir             string
	fileServer             http.Handler
	previewFileServer      http.Handler
	mcpHandler             http.Handler
	translationMu          sync.Mutex
	translationComparisons map[string]translationComparisonState
	modelCatalogMu         sync.Mutex
	modelCatalog           []translationModel
	modelCatalogAt         time.Time
}

func Serve(project string, options Options) error {
	if options.Host == "" {
		options.Host = "127.0.0.1"
	}
	if options.Port == 0 && !options.AutoPort {
		options.Port = 3000
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(options.Host, fmt.Sprint(options.Port)))
	if err != nil {
		return err
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	options.Port = actualPort
	server, err := NewServer(project, options)
	if err != nil {
		_ = listener.Close()
		return err
	}
	server.rebuild(false)
	go server.watch()
	localURL := fmt.Sprintf("http://localhost:%d", actualPort)
	fmt.Println("M-Press site:   " + localURL)
	if options.Workspace != nil {
		fmt.Println("M-Press workspace: " + localURL + "/__mpress/")
	}
	if server.token != "" {
		if server.requireWriteToken && options.Workspace != nil {
			fmt.Println("Remote workspace: " + localURL + "/__mpress/?token=" + server.token)
		} else if server.requireWriteToken {
			fmt.Println("Remote authoring: " + localURL + "/?token=" + server.token)
		}
		fmt.Println("M-Press MCP:    " + localURL + "/__mpress/mcp")
		fmt.Println("MCP token:      " + server.token)
	}
	if options.OpenBrowser {
		target := localURL + "/"
		if options.Contribution != nil {
			route := "/" + strings.Trim(options.Contribution.Route, "/")
			if route != "/" {
				route += "/"
			}
			target = localURL + route + "?contribute=1"
		}
		go func() {
			time.Sleep(120 * time.Millisecond)
			_ = openBrowser(target)
		}()
	}
	return http.Serve(listener, server.Handler())
}

func openBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}

func NewServer(project string, options Options) (*Server, error) {
	abs, err := filepath.Abs(project)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(abs)
	if err != nil {
		return nil, err
	}
	if options.Host == "" {
		options.Host = "127.0.0.1"
	}
	token := strings.TrimSpace(options.Token)
	requireWriteToken := token != "" || !isLoopbackHost(options.Host)
	if token == "" {
		token, err = newToken()
		if err != nil {
			return nil, err
		}
	}
	previewDir := filepath.Join(abs, ".mpress", "live-preview")
	if err := os.RemoveAll(previewDir); err != nil {
		return nil, err
	}
	s := &Server{
		project:                abs,
		cfg:                    cfg,
		options:                options,
		token:                  token,
		requireWriteToken:      requireWriteToken,
		events:                 newHub(),
		state:                  BuildState{Status: "starting"},
		previewDir:             previewDir,
		fileServer:             http.FileServer(http.Dir(cfg.OutputPath(abs))),
		previewFileServer:      http.FileServer(http.Dir(previewDir)),
		translationComparisons: map[string]translationComparisonState{},
	}
	s.mcpHandler = s.newMCPHandler()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/__mpress/events", s.events.handle)
	mux.HandleFunc("/__mpress/api/project", s.handleProject)
	mux.HandleFunc("/__mpress/api/contribution", s.handleContribution)
	mux.HandleFunc("/__mpress/api/open-file", s.handleOpenFile)
	mux.HandleFunc("/__mpress/api/setup/repository", s.handleRepositorySetup)
	mux.HandleFunc("/__mpress/api/tree", s.handleTree)
	mux.HandleFunc("/__mpress/api/file", s.handleFile)
	mux.HandleFunc("/__mpress/api/block", s.handleBlock)
	mux.HandleFunc("/__mpress/api/blocks", s.handleBlocks)
	mux.HandleFunc("/__mpress/api/import", s.handleImport)
	mux.HandleFunc("/__mpress/api/check", s.handleCheck)
	mux.HandleFunc("/__mpress/api/lighthouse", s.handleLighthouse)
	mux.HandleFunc("/__mpress/api/knowledge", s.handleKnowledge)
	mux.HandleFunc("/__mpress/api/export", s.handleExport)
	mux.HandleFunc("/__mpress/api/config", s.handleConfig)
	mux.HandleFunc("/__mpress/api/config-form", s.handleConfigForm)
	mux.HandleFunc("/__mpress/api/deployment", s.handleDeployment)
	mux.HandleFunc("/__mpress/api/onboarding", s.handleOnboarding)
	mux.HandleFunc("/__mpress/api/translations", s.handleTranslations)
	mux.HandleFunc("/__mpress/api/translation-models", s.handleTranslationModels)
	mux.HandleFunc("/__mpress/api/route", s.handleRoute)
	mux.HandleFunc("/__mpress/api/blog-image", s.handleBlogImage)
	mux.HandleFunc("/__mpress/api/blog-posts", s.handleBlogPosts)
	mux.HandleFunc("/__mpress/api/markdown-preview", s.handleMarkdownPreview)
	mux.HandleFunc("/__mpress/api/versions", s.handleVersions)
	mux.HandleFunc("/__mpress/api/brand-asset", s.handleBrandAsset)
	mux.HandleFunc("/__mpress/api/image-asset", s.handleImageAsset)
	mux.HandleFunc("/__mpress/api/preview", s.handlePreview)
	mux.Handle("/__mpress/mcp", s.authoriseMCP(s.mcpHandler))
	mux.HandleFunc("/__mpress/devbar.html", serveText("text/html; charset=utf-8", devbarHTML()))
	mux.HandleFunc("/__mpress/devbar.css", serveText("text/css; charset=utf-8", devbarCSS))
	mux.HandleFunc("/__mpress/devbar.js", serveText("text/javascript; charset=utf-8", devbarJS))
	if s.options.Workspace != nil {
		mux.HandleFunc("/__mpress/app.css", serveText("text/css; charset=utf-8", s.options.Workspace.CSS))
		mux.HandleFunc("/__mpress/app.js", serveText("text/javascript; charset=utf-8", s.options.Workspace.JS))
	}
	mux.HandleFunc("/__mpress/", s.handleWorkspace)
	mux.Handle("/__mpress/preview/", http.StripPrefix("/__mpress/preview", s.previewFileServer))
	mux.HandleFunc("/", s.handleSite)
	return securityHeaders(mux)
}

func (s *Server) handleKnowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if !s.cfg.Knowledge.Enabled {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "ready": false})
		return
	}
	store, err := knowledge.LoadAll(s.cfg.OutputPath(s.project))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "ready": false, "error": err.Error()})
		return
	}
	versions := make(map[string]bool)
	for _, page := range store.Pages {
		versions[page.Version] = true
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	results := []knowledge.SearchResult{}
	if query != "" {
		results = store.Search(knowledge.SearchOptions{Query: query, Language: strings.TrimSpace(r.URL.Query().Get("language")), Version: strings.TrimSpace(r.URL.Query().Get("version")), Limit: 8})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": true, "ready": true, "title": store.Manifest.Title,
		"digest": store.Manifest.Digest, "pages": len(store.Pages), "sections": len(store.Chunks),
		"languages": store.Manifest.Languages, "versions": len(versions), "query": query, "results": results,
	})
}

func (s *Server) handleTranslations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		language := strings.TrimSpace(r.URL.Query().Get("lang"))
		file := strings.TrimSpace(r.URL.Query().Get("file"))
		capabilities := translate.DetectModelCapabilities(s.cfg)
		selectionLanguage := language
		if selectionLanguage == "" {
			for _, candidate := range s.cfg.Site.Languages {
				if candidate != s.cfg.Site.DefaultLanguage {
					selectionLanguage = candidate
					break
				}
			}
		}
		draft, draftReady := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{
			Task: translate.TaskDraft, TargetLanguage: selectionLanguage, Capabilities: capabilities,
		})
		refinement, refinementReady := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{
			Task: translate.TaskRefinement, TargetLanguage: selectionLanguage, Capabilities: capabilities,
		})
		audit, auditReady := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{
			Task: translate.TaskAudit, TargetLanguage: selectionLanguage, PreviousProvider: refinement.Provider, Capabilities: capabilities,
		})
		response := map[string]any{
			"defaultLanguage": s.cfg.Site.DefaultLanguage,
			"languages":       s.cfg.Site.Languages,
			"languageLabels":  s.cfg.Site.LanguageLabels,
			"languageCatalog": standardTranslationLanguages(),
			"provider":        s.cfg.Translation.Provider,
			"model":           s.cfg.Translation.Model,
			"reasoningEffort": s.cfg.Translation.ReasoningEffort,
			"languageModels":  s.cfg.Translation.LanguageModels,
			"apiKeyEnv":       s.cfg.Translation.APIKeyEnv,
			"hasAPIKey":       translate.ResolveAPIKey(s.cfg) != "",
			"capabilities":    capabilities,
			"automatic": map[string]any{
				"ready":      draftReady,
				"draft":      map[string]any{"ready": draftReady, "selection": draft},
				"refinement": map[string]any{"ready": refinementReady, "selection": refinement},
				"audit":      map[string]any{"ready": auditReady, "selection": audit},
			},
		}
		contentDir := s.cfg.ContentPath(s.project)
		var markdownDocuments, mpdDocuments int
		_ = filepath.WalkDir(contentDir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return nil
			}
			extension := strings.ToLower(filepath.Ext(path))
			if extension == ".mpd" {
				mpdDocuments++
			} else if extension == ".md" || extension == ".markdown" {
				markdownDocuments++
			}
			return nil
		})
		response["documentFormats"] = map[string]any{
			"markdown":  markdownDocuments,
			"mpd":       mpdDocuments,
			"nativeMPD": markdownDocuments == 0 && mpdDocuments > 0,
		}
		targets := 0
		for _, candidate := range s.cfg.Site.Languages {
			if candidate != s.cfg.Site.DefaultLanguage {
				targets++
			}
		}
		if targets == 0 {
			response["report"] = translate.Report{}
			writeJSON(w, http.StatusOK, response)
			return
		}
		report, err := translate.NewEngine(s.project, s.cfg, nil).Run(r.Context(), translate.Options{
			Language: language, File: file, DryRun: true,
			Scope: r.URL.Query().Get("scope"), Force: r.URL.Query().Get("force") == "true",
		})
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		response["report"] = report
		writeJSON(w, http.StatusOK, response)
	case http.MethodPost:
		if !s.canWrite(w, r) {
			return
		}
		var input struct {
			Action   string `json:"action"`
			Language string `json:"language"`
			Label    string `json:"label"`
			File     string `json:"file"`
			Scope    string `json:"scope"`
			Force    bool   `json:"force"`
			Status   string `json:"status"`
			Workers  int    `json:"workers"`
			Refine   bool   `json:"refine"`
		}
		if err := decodeJSON(r, &input); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		if input.Action == "convert-mpd" {
			result, convertErr := projectconvert.Run(s.project, &s.cfg, "", "mpd")
			if convertErr != nil {
				writeAPIError(w, http.StatusBadRequest, convertErr)
				return
			}
			s.rebuild(true)
			writeJSON(w, http.StatusOK, map[string]any{"conversion": result, "state": s.currentState()})
			return
		}
		if input.Action == "add-language" {
			language := strings.TrimSpace(input.Language)
			standard, isStandard := standardLanguageChoice(language)
			if isStandard {
				language = standard.Code
			}
			if !validLanguageCode(language) {
				writeAPIError(w, http.StatusBadRequest, errors.New("language codes use letters, numbers and hyphens"))
				return
			}
			for _, existing := range s.cfg.Site.Languages {
				if strings.EqualFold(existing, language) {
					writeAPIError(w, http.StatusConflict, fmt.Errorf("language %q already exists", language))
					return
				}
			}
			updated := s.cfg
			updated.Site.Languages = append(append([]string{}, updated.Site.Languages...), language)
			labels := make(map[string]string, len(s.cfg.Site.LanguageLabels)+1)
			for code, display := range s.cfg.Site.LanguageLabels {
				labels[code] = display
			}
			updated.Site.LanguageLabels = labels
			label := strings.TrimSpace(input.Label)
			if label == "" {
				if standard.NativeName != "" {
					label = standard.NativeName
				} else if standard.Name != "" {
					label = standard.Name
				} else {
					label = language
				}
			}
			updated.Site.LanguageLabels[language] = label
			if err := updated.Validate(); err != nil {
				writeAPIError(w, http.StatusBadRequest, err)
				return
			}
			path := filepath.Join(s.project, config.Filename)
			current, err := os.ReadFile(path)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, err)
				return
			}
			data, err := setYAMLValue(current, []string{"site", "languages"}, updated.Site.Languages)
			if err == nil {
				data, err = setYAMLValue(data, []string{"site", "languageLabels"}, updated.Site.LanguageLabels)
			}
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, err)
				return
			}
			if err := s.writeSource(path, data, revision(current)); err != nil {
				writeAPIError(w, http.StatusConflict, err)
				return
			}
			s.cfg = updated
			s.rebuild(true)
			writeJSON(w, http.StatusOK, map[string]any{"language": language, "languages": updated.Site.Languages, "languageLabels": updated.Site.LanguageLabels, "state": s.currentState()})
			return
		}
		if input.Action == "mark" {
			report, markErr := translate.NewEngine(s.project, s.cfg, nil).Mark(input.Language, input.File, input.Status)
			if markErr != nil {
				writeAPIError(w, http.StatusBadRequest, markErr)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"report": translate.Report{Files: []translate.FileReport{report}, Segments: report.Segments}, "state": s.currentState()})
			return
		}
		if input.Action == "audit" {
			capabilities := translate.DetectModelCapabilities(s.cfg)
			draft, _ := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{Task: translate.TaskDraft, TargetLanguage: input.Language, Capabilities: capabilities})
			auditSelection, hasAudit := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{Task: translate.TaskAudit, TargetLanguage: input.Language, PreviousProvider: draft.Provider, Capabilities: capabilities})
			engine := translate.NewEngine(s.project, s.cfg, nil)
			var report translate.AuditReport
			var auditErr error
			if hasAudit {
				auditConfig := s.cfg
				translate.ApplyModelSelection(&auditConfig, auditSelection)
				reviewer, reviewerErr := translate.NewConfiguredAuditReviewer(auditConfig)
				if reviewerErr != nil {
					writeAPIError(w, http.StatusBadRequest, reviewerErr)
					return
				}
				report, auditErr = engine.AuditWithReviewer(r.Context(), input.Language, input.File, reviewer)
			} else {
				report, auditErr = engine.Audit(input.Language, input.File)
			}
			if auditErr != nil {
				writeAPIError(w, http.StatusBadRequest, auditErr)
				return
			}
			initialReport := report
			refinement := translate.RefinementReport{Language: input.Language}
			refinementSelection := translate.ModelSelection{}
			if input.Refine && len(report.Findings) > 0 {
				var hasRefinement bool
				refinementSelection, hasRefinement = translate.SelectBestTranslationModel(translate.ModelSelectionRequest{Task: translate.TaskRefinement, TargetLanguage: input.Language, Capabilities: capabilities})
				if hasRefinement {
					refinementConfig := s.cfg
					translate.ApplyModelSelection(&refinementConfig, refinementSelection)
					refinementEngine, providerErr := translate.NewConfiguredEngine(s.project, refinementConfig)
					if providerErr != nil {
						writeAPIError(w, http.StatusBadRequest, providerErr)
						return
					}
					refinement, auditErr = engine.RefineWithProvider(r.Context(), input.Language, input.File, report.Findings, refinementEngine.Provider)
					if auditErr != nil {
						writeAPIError(w, http.StatusBadGateway, auditErr)
						return
					}
					if refinement.Segments > 0 {
						finalAuditSelection, finalReady := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{Task: translate.TaskAudit, TargetLanguage: input.Language, PreviousProvider: refinementSelection.Provider, Capabilities: capabilities})
						if finalReady {
							finalConfig := s.cfg
							translate.ApplyModelSelection(&finalConfig, finalAuditSelection)
							finalReviewer, reviewerErr := translate.NewConfiguredAuditReviewer(finalConfig)
							if reviewerErr != nil {
								writeAPIError(w, http.StatusBadRequest, reviewerErr)
								return
							}
							report, auditErr = engine.AuditWithReviewer(r.Context(), input.Language, input.File, finalReviewer)
							auditSelection = finalAuditSelection
						} else {
							report, auditErr = engine.Audit(input.Language, input.File)
						}
						if auditErr != nil {
							writeAPIError(w, http.StatusBadRequest, auditErr)
							return
						}
					}
				}
			}
			if refinement.Written > 0 {
				s.rebuild(false)
			}
			writeJSON(w, http.StatusOK, map[string]any{"initialAudit": initialReport, "refinement": refinement, "refinementSelection": refinementSelection, "audit": report, "auditSelection": auditSelection, "state": s.currentState()})
			return
		}
		selectedConfig := s.cfg
		selection, selected := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{
			Task: translate.TaskDraft, TargetLanguage: input.Language, Capabilities: translate.DetectModelCapabilities(s.cfg),
		})
		if selected {
			translate.ApplyModelSelection(&selectedConfig, selection)
		}
		engine, err := translate.NewConfiguredEngine(s.project, selectedConfig)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		report, err := engine.Run(r.Context(), translate.Options{Language: input.Language, File: input.File, Scope: input.Scope, Force: input.Force, Workers: input.Workers})
		if err != nil {
			writeAPIError(w, http.StatusBadGateway, err)
			return
		}
		s.rebuild(false)
		writeJSON(w, http.StatusOK, map[string]any{"report": report, "selection": selection, "state": s.currentState()})
	default:
		methodNotAllowed(w)
	}
}

func validLanguageCode(value string) bool {
	if len(value) < 2 || len(value) > 24 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, char := range value {
		if char == '-' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return !strings.Contains(value, "--")
}

func serveText(contentType, value string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = io.WriteString(w, value)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/__mpress/" {
		http.NotFound(w, r)
		return
	}
	if s.options.Workspace != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, workspaceHTML(s.options.Workspace.HTML))
		return
	}
	returnPath := strings.TrimSpace(r.URL.Query().Get("return"))
	if returnPath == "" {
		returnPath = "/"
	}
	parsed, err := url.Parse(returnPath)
	if err != nil || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "/__mpress") {
		parsed = &url.URL{Path: "/"}
	}
	cleanPath := path.Clean(parsed.Path)
	if strings.HasSuffix(parsed.Path, "/") && cleanPath != "/" {
		cleanPath += "/"
	}
	request := r.Clone(r.Context())
	requestURL := *r.URL
	requestURL.Path = cleanPath
	requestURL.RawPath = ""
	requestURL.RawQuery = ""
	request.URL = &requestURL
	request.Header = r.Header.Clone()
	request.Header.Set("X-MPress-Workspace", "settings")
	s.handleSite(w, request)
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	payload := map[string]any{
		"name":          filepath.Base(s.project),
		"title":         s.cfg.Site.Title,
		"authoring":     s.options.Authoring,
		"requiresToken": s.requireWriteToken,
		"state":         s.currentState(),
		"onboarding":    s.onboarding(),
		"repository":    inspectRepository(s.project),
		"features": map[string]bool{
			"blog":         directoryContainsDocuments(filepath.Join(s.cfg.ContentPath(s.project), "blog")),
			"translations": len(s.cfg.Site.Languages) > 1,
			"versions":     s.cfg.Version.Enabled,
			"deployment":   len(s.cfg.Deploy.Targets) > 0,
		},
	}
	if s.options.Contribution != nil {
		payload["contribution"] = s.options.Contribution
		if guide := s.contributorGuide(); guide != nil {
			payload["contributorGuide"] = guide
		}
	}
	if s.options.Authoring && requestIsLoopback(r) {
		payload["token"] = s.token
	}
	writeJSON(w, http.StatusOK, payload)
}

func directoryContainsDocuments(directory string) bool {
	found := false
	_ = filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension == ".md" || extension == ".markdown" || extension == ".mpd" {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

type contributorGuide struct {
	Path     string `json:"path"`
	Markdown string `json:"markdown"`
}

func (s *Server) contributorGuide() *contributorGuide {
	candidates := []string{strings.TrimSpace(s.cfg.Contribution.Guide), "CONTRIBUTING.md", "CONTRIBUTORS.md", ".github/CONTRIBUTING.md"}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(strings.TrimSpace(candidate))
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		path, err := safeProjectPath(s.project, candidate)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxSourceSize {
			continue
		}
		return &contributorGuide{Path: candidate, Markdown: string(data)}
	}
	return nil
}

func safeProjectPath(project, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", errors.New("project paths must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("project path leaves the checkout")
	}
	return filepath.Join(project, clean), nil
}

type contributionFile struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

type contributionStatus struct {
	Branch      string             `json:"branch"`
	Files       []contributionFile `json:"files"`
	Diff        string             `json:"diff"`
	Clean       bool               `json:"clean"`
	Committed   bool               `json:"committed"`
	Pushed      bool               `json:"pushed"`
	CanOpenPR   bool               `json:"canOpenPR"`
	PullRequest string             `json:"pullRequest,omitempty"`
	Commit      string             `json:"commit,omitempty"`
	Ahead       int                `json:"ahead"`
	Fallback    []string           `json:"fallback"`
}

type contributionAction struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

func (s *Server) handleContribution(w http.ResponseWriter, r *http.Request) {
	if s.options.Contribution == nil {
		writeAPIError(w, http.StatusNotFound, errors.New("this development server is not a contribution checkout"))
		return
	}
	if !inspectRepository(s.project).IsGit {
		writeAPIError(w, http.StatusConflict, errors.New("the contribution checkout is not a Git repository"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		status, err := s.contributionStatus("")
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	case http.MethodPost:
		if !s.canWrite(w, r) {
			return
		}
		var input contributionAction
		if err := decodeJSON(r, &input); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		var pullRequest string
		var err error
		switch strings.ToLower(strings.TrimSpace(input.Action)) {
		case "commit":
			message := strings.TrimSpace(input.Message)
			if message == "" || len(message) > 160 {
				writeAPIError(w, http.StatusBadRequest, errors.New("write a commit message of 160 characters or fewer"))
				return
			}
			if err = runGit(s.project, "add", "--all"); err == nil {
				err = runGit(s.project, "commit", "-m", message)
			}
		case "push":
			err = runGit(s.project, "push", "--set-upstream", "origin", "HEAD")
			if err != nil {
				if _, lookupErr := exec.LookPath("gh"); lookupErr == nil {
					if strings.TrimSpace(gitOutput(s.project, "remote", "get-url", "mpress-fork")) == "" {
						command := exec.CommandContext(r.Context(), "gh", "repo", "fork", "--remote", "--remote-name", "mpress-fork")
						command.Dir = s.project
						if output, forkErr := command.CombinedOutput(); forkErr != nil {
							err = fmt.Errorf("the source repository rejected the push and GitHub could not prepare your fork: %s", strings.TrimSpace(string(output)))
						} else {
							err = runGit(s.project, "push", "--set-upstream", "mpress-fork", "HEAD")
						}
					} else {
						err = runGit(s.project, "push", "--set-upstream", "mpress-fork", "HEAD")
					}
				}
			}
		case "pull-request":
			if _, lookupErr := exec.LookPath("gh"); lookupErr != nil {
				err = errors.New("GitHub CLI is not available; use the copyable commands instead")
				break
			}
			title := strings.TrimSpace(input.Title)
			if title == "" || len(title) > 200 {
				writeAPIError(w, http.StatusBadRequest, errors.New("write a pull request title of 200 characters or fewer"))
				return
			}
			arguments := []string{"pr", "create", "--draft", "--title", title, "--body", strings.TrimSpace(input.Body)}
			command := exec.CommandContext(r.Context(), "gh", arguments...)
			command.Dir = s.project
			output, commandErr := command.CombinedOutput()
			if commandErr != nil {
				err = fmt.Errorf("GitHub could not open the draft pull request: %s", strings.TrimSpace(string(output)))
			} else {
				pullRequest = strings.TrimSpace(string(output))
			}
		default:
			writeAPIError(w, http.StatusBadRequest, errors.New("contribution action must be commit, push, or pull-request"))
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		status, statusErr := s.contributionStatus(pullRequest)
		if statusErr != nil {
			writeAPIError(w, http.StatusInternalServerError, statusErr)
			return
		}
		writeJSON(w, http.StatusOK, status)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) contributionStatus(pullRequest string) (contributionStatus, error) {
	info := inspectRepository(s.project)
	status := contributionStatus{
		Branch:      info.Branch,
		Fallback:    []string{"git add --all", `git commit -m "Describe the documentation change"`, "git push --set-upstream origin HEAD", "gh pr create --draft --fill"},
		PullRequest: pullRequest,
	}
	output, err := exec.Command("git", "-C", s.project, "status", "--short").Output()
	if err != nil {
		return status, fmt.Errorf("read contribution status: %w", err)
	}
	for _, line := range strings.Split(strings.TrimRight(string(output), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		status.Files = append(status.Files, contributionFile{Status: strings.TrimSpace(line[:2]), Path: strings.TrimSpace(line[3:])})
	}
	status.Clean = len(status.Files) == 0
	status.Commit = strings.TrimSpace(gitOutput(s.project, "rev-parse", "--short", "HEAD"))
	if start := strings.TrimSpace(s.options.Contribution.StartCommit); start != "" {
		fmt.Sscan(strings.TrimSpace(gitOutput(s.project, "rev-list", "--count", start+"..HEAD")), &status.Ahead)
	}
	upstream := strings.TrimSpace(gitOutput(s.project, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"))
	status.Pushed = upstream != "" && strings.TrimSpace(gitOutput(s.project, "rev-list", "--count", upstream+"..HEAD")) == "0"
	if !status.Pushed && info.Branch != "" {
		remote := strings.TrimSpace(gitOutput(s.project, "config", "--get", "branch."+info.Branch+".remote"))
		merge := strings.TrimSpace(gitOutput(s.project, "config", "--get", "branch."+info.Branch+".merge"))
		if remote != "" && remote != "." && merge != "" {
			output, remoteErr := exec.Command("git", "-C", s.project, "ls-remote", "--heads", remote, merge).Output()
			fields := strings.Fields(string(output))
			head := strings.TrimSpace(gitOutput(s.project, "rev-parse", "HEAD"))
			status.Pushed = remoteErr == nil && len(fields) >= 2 && fields[0] == head
		}
	}
	status.Committed = status.Clean && status.Ahead > 0
	_, ghErr := exec.LookPath("gh")
	remoteName := strings.TrimSpace(gitOutput(s.project, "config", "--get", "branch."+info.Branch+".remote"))
	remoteURL := strings.ToLower(strings.TrimSpace(gitOutput(s.project, "remote", "get-url", remoteName)))
	status.CanOpenPR = status.Pushed && ghErr == nil && (strings.Contains(remoteURL, "github.com/") || strings.Contains(remoteURL, "github.com:"))
	diff := contributionDiff(s.project)
	const maxDiff = 256 << 10
	if len(diff) > maxDiff {
		diff = diff[:maxDiff] + "\n\n[Diff truncated by M-Press]"
	}
	status.Diff = diff
	return status, nil
}

func contributionDiff(project string) string {
	unstaged, _ := exec.Command("git", "-C", project, "diff", "--no-ext-diff", "--").Output()
	staged, _ := exec.Command("git", "-C", project, "diff", "--cached", "--no-ext-diff", "--").Output()
	return string(staged) + string(unstaged)
}

func runGit(project string, args ...string) error {
	commandArgs := append([]string{"-C", project}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return errors.New(message)
	}
	return nil
}

type openFilePayload struct {
	Path string `json:"path"`
}

func (s *Server) handleOpenFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var payload openFilePayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	target, err := s.sourcePath(payload.Path, false)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	command, label, err := editorCommand(target, exec.LookPath)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	if err := command.Start(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Errorf("open source file: %w", err))
		return
	}
	go func() { _ = command.Wait() }()
	writeJSON(w, http.StatusOK, map[string]string{"editor": label, "path": payload.Path})
}

func editorCommand(target string, lookPath func(string) (string, error)) (*exec.Cmd, string, error) {
	for _, configured := range []string{os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		fields := strings.Fields(configured)
		if len(fields) == 0 {
			continue
		}
		binary, err := lookPath(fields[0])
		if err == nil {
			return exec.Command(binary, append(fields[1:], target)...), fields[0], nil
		}
	}
	candidates := []string{"code", "codium", "zed", "subl"}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "open")
	} else if runtime.GOOS != "windows" {
		candidates = append(candidates, "xdg-open")
	}
	for _, candidate := range candidates {
		if binary, err := lookPath(candidate); err == nil {
			return exec.Command(binary, target), candidate, nil
		}
	}
	return nil, "", errors.New("no supported editor was found; copy the source path instead")
}

type repositorySetupPayload struct {
	Action      string `json:"action"`
	Repository  string `json:"repository"`
	Destination string `json:"destination"`
	Branch      string `json:"branch"`
}

type repositoryInfo struct {
	Path         string   `json:"path"`
	IsGit        bool     `json:"isGit"`
	Origin       string   `json:"origin,omitempty"`
	Branch       string   `json:"branch,omitempty"`
	Dirty        bool     `json:"dirty"`
	GitHubCLI    bool     `json:"githubCLI"`
	OnWorkBranch bool     `json:"onWorkBranch"`
	ChangedFiles []string `json:"changedFiles,omitempty"`
}

func (s *Server) handleRepositorySetup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, inspectRepository(s.project))
	case http.MethodPost:
		if !s.canWrite(w, r) {
			return
		}
		var payload repositorySetupPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		payload.Action = strings.ToLower(strings.TrimSpace(payload.Action))
		payload.Branch = strings.TrimSpace(payload.Branch)
		if payload.Branch == "" {
			payload.Branch = "docs/mpress"
		}
		checkout := s.project
		restartRequired := false
		switch payload.Action {
		case "current":
			if !inspectRepository(checkout).IsGit {
				writeAPIError(w, http.StatusBadRequest, errors.New("the current project is not a Git checkout; clone or fork a repository first"))
				return
			}
		case "clone":
			var err error
			checkout, err = translate.CloneProject(r.Context(), payload.Repository, "", payload.Destination)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, err)
				return
			}
			restartRequired = true
		case "fork":
			var err error
			checkout, err = forkRepository(r.Context(), payload.Repository, payload.Destination)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, err)
				return
			}
			restartRequired = true
		default:
			writeAPIError(w, http.StatusBadRequest, errors.New("setup action must be current, clone, or fork"))
			return
		}
		if err := createWorkBranch(r.Context(), checkout, payload.Branch); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"repository":      inspectRepository(checkout),
			"restartRequired": restartRequired,
			"checkout":        checkout,
			"command":         fmt.Sprintf("mpress dev %q", checkout),
		})
	default:
		methodNotAllowed(w)
	}
}

func inspectRepository(project string) repositoryInfo {
	info := repositoryInfo{Path: project}
	_, ghErr := exec.LookPath("gh")
	info.GitHubCLI = ghErr == nil
	if _, err := os.Stat(filepath.Join(project, ".git")); err != nil {
		return info
	}
	info.IsGit = true
	info.Branch = strings.TrimSpace(gitOutput(project, "branch", "--show-current"))
	info.Origin = strings.TrimSpace(gitOutput(project, "remote", "get-url", "origin"))
	status := strings.TrimSpace(gitOutput(project, "status", "--porcelain"))
	info.Dirty = status != ""
	for _, line := range strings.Split(status, "\n") {
		if len(line) < 4 {
			continue
		}
		name := strings.TrimSpace(line[3:])
		if arrow := strings.LastIndex(name, " -> "); arrow >= 0 {
			name = name[arrow+4:]
		}
		if name != "" {
			info.ChangedFiles = append(info.ChangedFiles, name)
		}
	}
	branch := strings.ToLower(info.Branch)
	info.OnWorkBranch = branch != "" && branch != "main" && branch != "master" && branch != "trunk"
	return info
}

func gitOutput(project string, args ...string) string {
	commandArgs := append([]string{"-C", project}, args...)
	output, err := exec.Command("git", commandArgs...).Output()
	if err != nil {
		return ""
	}
	return string(output)
}

func createWorkBranch(ctx context.Context, project, branch string) error {
	return translate.PrepareWorkBranch(ctx, project, branch)
}

func forkRepository(ctx context.Context, repository, destination string) (string, error) {
	repository = strings.TrimSpace(repository)
	destination = strings.TrimSpace(destination)
	if repository == "" || destination == "" {
		return "", errors.New("repository and checkout directory are required")
	}
	if parsed, parseErr := url.Parse(repository); parseErr == nil && parsed.User != nil {
		return "", errors.New("repository URLs must not contain credentials; use the Git credential helper or gh")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return "", errors.New("GitHub CLI is required to create a fork; install gh or choose Clone")
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Stat(abs); statErr == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("checkout destination %s already exists", abs)
		}
		entries, readErr := os.ReadDir(abs)
		if readErr != nil {
			return "", readErr
		}
		if len(entries) != 0 {
			return "", fmt.Errorf("checkout destination %s is not empty", abs)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	command := exec.CommandContext(ctx, "gh", "repo", "fork", repository, "--clone", "--", abs)
	if _, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("GitHub could not create and clone the fork: %w", err)
	}
	if _, err := os.Stat(filepath.Join(abs, config.Filename)); err != nil {
		return "", fmt.Errorf("forked repository does not contain %s", config.Filename)
	}
	return abs, nil
}

func requestIsLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

type treeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Kind     string     `json:"kind"`
	Children []treeNode `json:"children,omitempty"`
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	var nodes []treeNode
	for _, name := range []string{s.cfg.Build.ContentDir, s.cfg.Build.StaticDir, config.Filename} {
		path := filepath.Join(s.project, filepath.FromSlash(name))
		if _, err := os.Stat(path); err != nil {
			continue
		}
		node, err := s.readTree(path)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		nodes = append(nodes, node)
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) readTree(path string) (treeNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return treeNode{}, err
	}
	rel, _ := filepath.Rel(s.project, path)
	node := treeNode{Name: info.Name(), Path: filepath.ToSlash(rel), Kind: "file"}
	if !info.IsDir() {
		return node, nil
	}
	node.Kind = "directory"
	entries, err := os.ReadDir(path)
	if err != nil {
		return node, err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" || entry.Name() == ".mpress" || entry.Name() == s.cfg.Build.OutputDir {
			continue
		}
		child, err := s.readTree(filepath.Join(path, entry.Name()))
		if err != nil {
			return node, err
		}
		node.Children = append(node.Children, child)
	}
	sort.SliceStable(node.Children, func(i, j int) bool {
		if node.Children[i].Kind != node.Children[j].Kind {
			return node.Children[i].Kind == "directory"
		}
		return strings.ToLower(node.Children[i].Name) < strings.ToLower(node.Children[j].Name)
	})
	return node, nil
}

type filePayload struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Revision string `json:"revision"`
	Route    string `json:"route,omitempty"`
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		path, err := s.sourcePath(r.URL.Query().Get("path"), false)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err)
			return
		}
		if len(data) > maxSourceSize {
			writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("file is too large to edit"))
			return
		}
		rel, _ := filepath.Rel(s.project, path)
		payload := filePayload{Path: filepath.ToSlash(rel), Content: string(data), Revision: revision(data)}
		payload.Route = s.routeFor(payload.Path, payload.Content)
		writeJSON(w, http.StatusOK, payload)
	case http.MethodPut:
		if !s.canWrite(w, r) {
			return
		}
		var payload filePayload
		if err := decodeJSON(r, &payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		path, err := s.sourcePath(payload.Path, true)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		if len(payload.Content) > maxSourceSize {
			writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("file is too large to edit"))
			return
		}
		if err := s.writeSource(path, []byte(payload.Content), payload.Revision); err != nil {
			if errors.Is(err, errRevisionConflict) {
				writeAPIError(w, http.StatusConflict, err)
			} else {
				writeAPIError(w, http.StatusBadRequest, err)
			}
			return
		}
		s.rebuild(true)
		data, _ := os.ReadFile(path)
		payload.Revision = revision(data)
		payload.Route = s.routeFor(payload.Path, payload.Content)
		writeJSON(w, http.StatusOK, map[string]any{"file": payload, "state": s.currentState()})
	default:
		methodNotAllowed(w)
	}
}

type previewPayload struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Revision string `json:"revision"`
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var payload previewPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	if len(payload.Content) > maxSourceSize {
		writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("file is too large to preview"))
		return
	}
	path, err := s.sourcePath(payload.Path, true)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" && ext != ".markdown" {
		writeAPIError(w, http.StatusBadRequest, errors.New("live preview is available for Markdown files"))
		return
	}
	current, err := os.ReadFile(path)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	if payload.Revision == "" || revision(current) != payload.Revision {
		writeAPIError(w, http.StatusConflict, errRevisionConflict)
		return
	}
	contentRel, err := filepath.Rel(s.cfg.ContentPath(s.project), path)
	if err != nil || contentRel == ".." || strings.HasPrefix(contentRel, ".."+string(filepath.Separator)) {
		writeAPIError(w, http.StatusBadRequest, errors.New("live preview source must be inside the content directory"))
		return
	}
	select {
	case <-r.Context().Done():
		return
	default:
	}
	s.previewMu.Lock()
	defer s.previewMu.Unlock()
	select {
	case <-r.Context().Done():
		return
	default:
	}
	buildID := revision([]byte(payload.Path + "\x00" + payload.Content))
	buildDir := filepath.Join(s.previewDir, buildID)
	result, buildErr := site.Build(s.project, site.BuildOptions{
		Strict: true, IncludeDrafts: true, OutputDir: buildDir,
		SourceOverrides: map[string]string{filepath.ToSlash(contentRel): payload.Content},
	})
	if buildErr != nil {
		_ = os.RemoveAll(buildDir)
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": buildErr.Error(), "diagnostics": result.Diagnostics})
		return
	}
	entries, _ := os.ReadDir(s.previewDir)
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != buildID {
			_ = os.RemoveAll(filepath.Join(s.previewDir, entry.Name()))
		}
	}
	route := s.routeFor(payload.Path, payload.Content)
	previewURL := "/__mpress/preview/" + buildID + "/"
	if route != "/" {
		previewURL += strings.TrimPrefix(route, "/")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "url": previewURL, "route": route,
		"pages": result.Pages, "files": result.Files, "durationMs": result.Duration.Milliseconds(), "diagnostics": result.Diagnostics,
	})
}

type blockPayload struct {
	Path     string `json:"path"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Markdown string `json:"markdown"`
	Revision string `json:"revision"`
}

func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var payload blockPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	path, err := s.sourcePath(payload.Path, true)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	if payload.Revision != revision(data) {
		writeAPIError(w, http.StatusConflict, errRevisionConflict)
		return
	}
	if payload.Start < 0 || payload.End < payload.Start || payload.End > len(data) {
		writeAPIError(w, http.StatusBadRequest, errors.New("invalid source range"))
		return
	}
	updated := append([]byte{}, data[:payload.Start]...)
	updated = append(updated, []byte(payload.Markdown)...)
	updated = append(updated, data[payload.End:]...)
	if err := s.writeSource(path, updated, payload.Revision); err != nil {
		writeAPIError(w, http.StatusConflict, err)
		return
	}
	s.rebuild(true)
	writeJSON(w, http.StatusOK, map[string]any{"revision": revision(updated), "state": s.currentState()})
}

func (s *Server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	path, err := s.sourcePath(r.URL.Query().Get("path"), false)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	rel, _ := filepath.Rel(s.cfg.ContentPath(s.project), path)
	blocks := editableBlocks(filepath.ToSlash(rel), string(data))
	writeJSON(w, http.StatusOK, map[string]any{"revision": revision(data), "blocks": blocks})
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	if err := r.ParseMultipartForm(maxSourceSize); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("choose a Markdown file"))
		return
	}
	defer file.Close()
	name := safeImportName(header)
	if name == "" {
		writeAPIError(w, http.StatusBadRequest, errors.New("only Markdown files can be imported"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxSourceSize+1))
	if err != nil || len(data) > maxSourceSize {
		writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("file is too large to import"))
		return
	}
	target := filepath.Join(s.cfg.ContentPath(s.project), name)
	existingRevision := ""
	if existing, err := os.ReadFile(target); err == nil {
		if r.FormValue("replaceStarter") != "true" || !s.onboarding() {
			writeAPIError(w, http.StatusConflict, fmt.Errorf("%s already exists", name))
			return
		}
		existingRevision = revision(existing)
	}
	if err := s.writeSource(target, data, existingRevision); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	s.rebuild(true)
	rel, _ := filepath.Rel(s.project, target)
	writeJSON(w, http.StatusCreated, map[string]any{"path": filepath.ToSlash(rel), "state": s.currentState()})
}

func safeImportName(header *multipart.FileHeader) string {
	name := filepath.Base(header.Filename)
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".md" && ext != ".markdown" {
		return ""
	}
	return name
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	result, checkErr := operations.Check(s.project, true)
	state := BuildState{Status: "ready", Pages: result.Build.Pages, Files: result.Build.Files, DurationMS: result.Build.Duration.Milliseconds(), Diagnostics: result.Build.Diagnostics, BuiltAt: time.Now(), Revision: s.currentState().Revision + 1}
	if checkErr != nil && len(result.Broken) == 0 {
		state.Status = "error"
		state.Error = checkErr.Error()
	}
	s.setState(state)
	writeJSON(w, http.StatusOK, map[string]any{"ok": checkErr == nil, "state": state, "broken": result.Broken, "performance": result.Performance})
}

func (s *Server) handleLighthouse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var input struct {
		Path       string `json:"path"`
		FormFactor string `json:"formFactor"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		path = "/"
	}
	parsed, err := url.Parse(path)
	if err != nil || !strings.HasPrefix(path, "/") || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(path, "//") {
		writeAPIError(w, http.StatusBadRequest, errors.New("Lighthouse can test only a page in this development site"))
		return
	}
	target := "http://" + r.Host + parsed.EscapedPath()
	if parsed.RawQuery != "" {
		target += "?" + parsed.RawQuery
	}
	result, err := lighthouse.Run(r.Context(), target, lighthouse.Options{FormFactor: input.FormFactor})
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var input struct {
		Strict bool `json:"strict"`
		Drafts bool `json:"drafts"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	temporary, err := os.CreateTemp("", "mpress-download-*.zip")
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	archivePath := temporary.Name()
	_ = temporary.Close()
	_ = os.Remove(archivePath)
	defer os.Remove(archivePath)

	s.buildMu.Lock()
	result, err := exportzip.Create(s.project, archivePath, exportzip.Options{Strict: input.Strict, IncludeDrafts: input.Drafts})
	s.buildMu.Unlock()
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	archive, err := os.Open(archivePath)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	defer archive.Close()
	info, err := archive.Stat()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	filename := exportzip.DefaultFilename(s.project)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprint(info.Size()))
	w.Header().Set("X-MPress-Filename", filename)
	w.Header().Set("X-MPress-Pages", fmt.Sprint(result.Pages))
	w.Header().Set("X-MPress-Files", fmt.Sprint(result.Files))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, archive)
}

type configPayload struct {
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	BaseURL         string         `json:"baseURL"`
	ColorScheme     string         `json:"colorScheme"`
	AccentColor     string         `json:"accentColor"`
	HoverColor      string         `json:"hoverColor"`
	HoverColorLight string         `json:"hoverColorLight"`
	HoverColorDark  string         `json:"hoverColorDark"`
	Config          *config.Config `json:"config,omitempty"`
	Complete        bool           `json:"complete,omitempty"`
	Revision        string         `json:"revision"`
}

// handleConfigForm serves quick setup and complete settings fragments from the
// same text-first Markdown form grammar available to site authors. Values are
// hydrated by the development client after insertion, so this endpoint remains
// a stable presentation template instead of duplicating the configuration
// object.
func (s *Server) handleConfigForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	section := r.URL.Query().Get("section")
	source := `@form[config.quick]{id="config-form" class="form-grid config-form" autocomplete="off"}

## Site identity

Site title*: [text](title)

Description: [textarea](description){rows="4"}

Public URL: [url](baseURL){placeholder="https://docs.example.com"}

## Appearance

Colour scheme: [select](colorScheme)
- Use the device setting = system
- Light = light
- Dark = dark

Accent colour: [color](accentColor)

Light-mode hover: [color](hoverColorLight)

Dark-mode hover: [color](hoverColorDark)

@end
`
	sections := map[string]string{
		"site": `@form[config.site]{id="config-site-form" autocomplete="off"}

Site title*: [text](site.title)

Description: [textarea](site.description){rows="4"}

Public URL: [url](site.baseURL){placeholder="https://docs.example.com"}

Accessibility shortcut: [shortcut](accessibility.shortcut){placeholder="Mod+A"}

@end
`,
		"languages": `@form[config.languages]{id="config-languages-form" autocomplete="off"}

Default language*: [select](site.defaultLanguage)
{{LANGUAGE_OPTIONS}}

Missing translation: [select](site.missingTranslation)
- Link to default language = link-to-default
- Omit the page = omit

[ ] Default language at the root (site.defaultLanguageAtRoot)

@end
`,
		"brand": `@form[config.brand]{id="config-brand-form" autocomplete="off"}

Light-mode logo: [file](site.logoLightUpload){accept="image/svg+xml,image/png,image/jpeg,image/webp,image/avif"}

Hidden light-mode logo path: [hidden](site.logoLight)

Dark-mode logo: [file](site.logoDarkUpload){accept="image/svg+xml,image/png,image/jpeg,image/webp,image/avif"}

Hidden dark-mode logo path: [hidden](site.logoDark)

Favicon: [file](site.faviconUpload){accept="image/svg+xml,image/png,image/jpeg,image/webp,image/avif"}

Hidden favicon path: [hidden](site.favicon)

Social sharing image: [file](site.socialImageUpload){accept="image/svg+xml,image/png,image/jpeg,image/webp,image/avif"}

Hidden social sharing image path: [hidden](site.socialImage)

@end
`,
		"social": `@form[config.social]{id="config-social-form" autocomplete="off"}

GitHub URL: [url](social.github)

Discord URL: [url](social.discord)

Reddit URL: [url](social.reddit)

X URL: [url](social.x)

RSS URL: [text](social.rss){placeholder="/blog/rss.xml" autocomplete="url"}

Sponsor URL: [url](social.sponsor)

[ ] Let readers contribute (contribution.enabled)

Git repository: [url](contribution.repository){placeholder="https://github.com/owner/docs.git"}

Source branch: [text](contribution.branch){placeholder="main"}

Contributor guide: [text](contribution.guide){placeholder="CONTRIBUTING.md"}

[ ] Enable browser quick edit (contribution.quickEdit)

Edit page URL base (optional): [text](social.editURL){placeholder="https://github.com/owner/docs/edit/main/content"}

@end
`,
		"build": `@form[config.build]{id="config-build-form" autocomplete="off"}

Content directory*: [text](build.contentDir)

Static directory*: [text](build.staticDir)

Output directory*: [text](build.outputDir)

Navigation file*: [text](build.navFile)

Custom CSS file: [text](build.customCSS)

Version artifacts directory*: [text](versioning.artifactsDir)

[x] Generate the MCP knowledge base (knowledge.enabled)

@end
`,
		"blog": `@form[config.blog]{id="config-blog-form" autocomplete="off"}

Landing page style: [select](blog.landingStyle)
- Feature the latest article = featured
- Equal card grid = grid
- Compact article list = list

Tagline: [textarea](blog.tagline){rows="2" placeholder="News, releases, and engineering stories."}

[ ] Show tags by default (blog.showTags)

Hero heading size: [select](blog.headingSize)
- Standard = default
- Compact = compact
- Large = large

Hero image mode: [select](blog.imageMode)
- Framed panel = panel
- Floating image = floating

Hero image fit: [select](blog.imageFit)
- Fill the frame = cover
- Show the complete image = contain

Floating image width: [text](blog.imageWidth){placeholder="100"}

Image background: [color](blog.imageBackground)

@end
`,
		"theme": `@form[config.theme]{id="config-theme-form" autocomplete="off"}

## Colour Scheme

Default Scheme: [select](theme.colorScheme)
- Use the device setting = system
- Light = light
- Dark = dark

Accent colour: [color](theme.accentColor)

Light-mode hover: [color](theme.hoverColorLight)

Dark-mode hover: [color](theme.hoverColorDark)

## Search

[ ] Enable search (search.enabled)

[ ] Remember recent searches (search.rememberRecent)

Search prompt: [text](search.placeholder){placeholder="Search documentation" maxlength="80"}

Results shown: [select](search.maxResults)
- 6 results = 6
- 12 results = 12
- 18 results = 18
- 24 results = 24

Search shortcut: [shortcut](search.shortcut){placeholder="Mod+K"}

@end
`,
		"accessibility": `@form[config.accessibility]{id="config-accessibility-form" autocomplete="off"}

[ ] Enable the accessibility menu (accessibility.enabled)

@end
`,
		"layout": `@form[config.layout]{id="config-layout-form" autocomplete="off"}

Layout preset: [select](theme.layout.preset)
- Starlight, balanced reading width = starlight
- Wide, more room for dense reference pages = wide
- Reading, a narrower prose measure = reading
- Custom = custom

Article width*: [text](theme.layout.contentWidth){placeholder="45rem"}

Wide-page width*: [text](theme.layout.wideContentWidth){placeholder="64rem"}

Navigation width*: [text](theme.layout.sidebarWidth){placeholder="18.75rem"}

Table of contents width*: [text](theme.layout.tocWidth){placeholder="16rem"}

Article-to-TOC gap*: [text](theme.layout.contentTocGap){placeholder="2.5rem"}

Desktop alignment: [select](theme.layout.alignment)
- Centre article and TOC together = cluster
- Align after navigation = left

Table of contents: [select](theme.layout.toc)
- Show on the right = right
- Hide = hidden

@end
`,
		"versioning": `@form[config.versioning]{id="config-versioning-form" autocomplete="off"}

[ ] Enable versioning (versioning.enabled)

@end
`,
		"translation": `@form[config.translation]{id="config-translation-form" autocomplete="off"}

Provider: [select](translation.provider)
- OpenRouter = openrouter
- OpenAI = openai
- OpenAI-compatible = openai-compatible
- Local Codex = codex
- Local Claude Code = claude

Model: [text](translation.model){placeholder="openai/gpt-5-mini"}

Reasoning effort: [select](translation.reasoningEffort)
- Provider default =
- Disabled = none
- Minimal = minimal
- Low = low
- Medium = medium
- High = high
- Extra high = xhigh
- Maximum = max

Local command (optional): [text](translation.command){placeholder="codex or claude"}

API base URL: [url](translation.baseURL)

API key environment variable: [text](translation.apiKeyEnv)

Source language: [text](translation.sourceLanguage)

Glossary file: [text](translation.glossary){placeholder="glossary.yaml"}

Style guide file: [text](translation.styleGuide){placeholder="docs/writing-standard.md"}

State directory: [text](translation.stateDir)

Input price per million tokens (USD): [number](translation.inputPricePerMillion){min=0 step=0.000001}

Output price per million tokens (USD): [number](translation.outputPricePerMillion){min=0 step=0.000001}

OpenRouter data policy: [select](translation.dataCollection)
- Deny data collection = deny
- Allow data collection = allow
- Provider default {value=}

[ ] Require structured-output support (translation.requireParameters)

@end
`,
		"deploy": `@form[config.deploy]{id="config-deploy-form" autocomplete="off"}

Default target: [text](deploy.default){placeholder="production"}

		@end
`,
	}
	languageOptions := make([]string, 0, len(s.cfg.Site.Languages))
	for _, code := range s.cfg.Site.Languages {
		label := strings.TrimSpace(s.cfg.Site.LanguageLabels[code])
		if label == "" {
			label = code
		}
		label = strings.NewReplacer("\n", " ", "\r", " ", "=", "-").Replace(label)
		languageOptions = append(languageOptions, "- "+label+" ("+code+") = "+code)
	}
	if len(languageOptions) == 0 {
		languageOptions = append(languageOptions, "- "+s.cfg.Site.DefaultLanguage+" = "+s.cfg.Site.DefaultLanguage)
	}
	sections["languages"] = strings.Replace(sections["languages"], "{{LANGUAGE_OPTIONS}}", strings.Join(languageOptions, "\n"), 1)
	if section != "" {
		var ok bool
		source, ok = sections[section]
		if !ok {
			writeAPIError(w, http.StatusBadRequest, fmt.Errorf("unknown configuration form section %q", section))
			return
		}
	}
	page, diagnostics, err := content.NewRenderer().Parse(".mpress/config-form.md", "en", source)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			writeAPIError(w, http.StatusInternalServerError, errors.New(diagnostic.Message))
			return
		}
	}
	rendered := page.HTML
	if section != "" {
		// The complete settings editor owns the submit event. Reuse the parser's
		// field markup as a fragment so this page does not create a nested form.
		start := strings.Index(rendered, ">")
		end := strings.LastIndex(rendered, "</form>")
		if start < 0 || end <= start {
			writeAPIError(w, http.StatusInternalServerError, errors.New("site settings form did not render a form element"))
			return
		}
		rendered = rendered[start+1 : end]
	}
	writeJSON(w, http.StatusOK, map[string]string{"html": rendered})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.project, config.Filename)
	current, err := os.ReadFile(path)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		complete := s.cfg
		writeJSON(w, http.StatusOK, configPayload{Title: s.cfg.Site.Title, Description: s.cfg.Site.Description, BaseURL: s.cfg.Site.BaseURL, ColorScheme: s.cfg.Theme.ColorScheme, AccentColor: s.cfg.Theme.AccentColor, HoverColor: s.cfg.Theme.HoverColor, HoverColorLight: s.cfg.Theme.HoverColorLight, HoverColorDark: s.cfg.Theme.HoverColorDark, Config: &complete, Revision: revision(current)})
	case http.MethodPut:
		if !s.canWrite(w, r) {
			return
		}
		var payload configPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		if payload.Complete {
			if payload.Config == nil {
				writeAPIError(w, http.StatusBadRequest, errors.New("complete configuration is required"))
				return
			}
			updated := *payload.Config
			if err := updated.Validate(); err != nil {
				writeAPIError(w, http.StatusBadRequest, err)
				return
			}
			data, err := yaml.Marshal(updated)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, err)
				return
			}
			if err := s.writeSource(path, data, payload.Revision); err != nil {
				status := http.StatusBadRequest
				if errors.Is(err, errRevisionConflict) {
					status = http.StatusConflict
				}
				writeAPIError(w, status, err)
				return
			}
			s.cfg = updated
			s.rebuild(true)
			writeJSON(w, http.StatusOK, map[string]any{"revision": revision(data), "state": s.currentState()})
			return
		}
		updated := s.cfg
		updated.Site.Title = strings.TrimSpace(payload.Title)
		updated.Site.Description = strings.TrimSpace(payload.Description)
		updated.Site.BaseURL = strings.TrimSpace(payload.BaseURL)
		updated.Theme.ColorScheme = strings.TrimSpace(payload.ColorScheme)
		updated.Theme.AccentColor = strings.TrimSpace(payload.AccentColor)
		updated.Theme.HoverColor = strings.TrimSpace(payload.HoverColor)
		updated.Theme.HoverColorLight = strings.TrimSpace(payload.HoverColorLight)
		updated.Theme.HoverColorDark = strings.TrimSpace(payload.HoverColorDark)
		if updated.Theme.ColorScheme != "system" && updated.Theme.ColorScheme != "light" && updated.Theme.ColorScheme != "dark" {
			writeAPIError(w, http.StatusBadRequest, errors.New("colour scheme must be system, light or dark"))
			return
		}
		if err := updated.Validate(); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		data := append([]byte{}, current...)
		for _, change := range []struct {
			path  []string
			value any
		}{
			{[]string{"site", "title"}, updated.Site.Title},
			{[]string{"site", "description"}, updated.Site.Description},
			{[]string{"site", "baseURL"}, updated.Site.BaseURL},
			{[]string{"theme", "colorScheme"}, updated.Theme.ColorScheme},
			{[]string{"theme", "accentColor"}, updated.Theme.AccentColor},
			{[]string{"theme", "hoverColor"}, updated.Theme.HoverColor},
			{[]string{"theme", "hoverColorLight"}, updated.Theme.HoverColorLight},
			{[]string{"theme", "hoverColorDark"}, updated.Theme.HoverColorDark},
		} {
			data, err = setYAMLValue(data, change.path, change.value)
			if err != nil {
				break
			}
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		if err := s.writeSource(path, data, payload.Revision); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, errRevisionConflict) {
				status = http.StatusConflict
			}
			writeAPIError(w, status, err)
			return
		}
		s.cfg = updated
		s.rebuild(true)
		writeJSON(w, http.StatusOK, map[string]any{"revision": revision(data), "state": s.currentState()})
	default:
		methodNotAllowed(w)
	}
}

type deploymentPayload struct {
	Provider         string `json:"provider"`
	Target           string `json:"target"`
	AccountID        string `json:"accountID"`
	Project          string `json:"project"`
	ProductionBranch string `json:"productionBranch"`
	Domain           string `json:"domain"`
	Environment      string `json:"environment"`
}

func (s *Server) handleDeployment(w http.ResponseWriter, r *http.Request) {
	targetName := s.cfg.Deploy.Default
	if targetName == "" {
		targetName = "cloudflare"
	}
	target := s.cfg.Deploy.Targets[targetName]
	switch r.Method {
	case http.MethodGet:
		provider := target.Provider
		if provider == "" {
			provider = "cloudflare-pages"
		}
		writeJSON(w, http.StatusOK, map[string]any{"name": targetName, "provider": provider, "target": target, "hasCloudflareToken": strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN")) != "", "hasNetlifyToken": strings.TrimSpace(os.Getenv("NETLIFY_AUTH_TOKEN")) != ""})
	case http.MethodPut:
		if !s.canWrite(w, r) {
			return
		}
		var payload deploymentPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		payload.Provider = strings.TrimSpace(payload.Provider)
		if payload.Provider == "" {
			payload.Provider = "cloudflare-pages"
		}
		payload.Target = strings.TrimSpace(payload.Target)
		if payload.Target == "" {
			payload.Target = targetName
		}
		payload.Project = strings.TrimSpace(payload.Project)
		if payload.Project == "" {
			writeAPIError(w, http.StatusBadRequest, errors.New("a deployment project or site is required"))
			return
		}
		if payload.Provider == "cloudflare-pages" && strings.Trim(payload.Project, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
			writeAPIError(w, http.StatusBadRequest, errors.New("Cloudflare project names use lowercase letters, numbers and hyphens"))
			return
		}
		if payload.Provider != "cloudflare-pages" && payload.Provider != "netlify" {
			writeAPIError(w, http.StatusBadRequest, errors.New("deployment provider must be Cloudflare Pages or Netlify"))
			return
		}
		updated := s.cfg
		if updated.Deploy.Targets == nil {
			updated.Deploy.Targets = map[string]config.DeployTarget{}
		}
		updated.Deploy.Default = payload.Target
		updated.Deploy.Targets[payload.Target] = config.DeployTarget{Provider: payload.Provider, AccountID: strings.TrimSpace(payload.AccountID), Project: payload.Project, ProductionBranch: strings.TrimSpace(payload.ProductionBranch), Domain: strings.TrimSpace(payload.Domain)}
		if updated.Deploy.Targets[payload.Target].ProductionBranch == "" {
			t := updated.Deploy.Targets[payload.Target]
			t.ProductionBranch = "main"
			updated.Deploy.Targets[payload.Target] = t
		}
		if err := updated.Validate(); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		current, err := os.ReadFile(filepath.Join(s.project, config.Filename))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		data, err := setYAMLValue(current, []string{"deploy"}, updated.Deploy)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		if err := s.writeSource(filepath.Join(s.project, config.Filename), data, revision(current)); err != nil {
			writeAPIError(w, http.StatusConflict, err)
			return
		}
		s.cfg = updated
		writeJSON(w, http.StatusOK, map[string]any{"name": payload.Target, "provider": payload.Provider, "target": updated.Deploy.Targets[payload.Target], "hasCloudflareToken": strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN")) != "", "hasNetlifyToken": strings.TrimSpace(os.Getenv("NETLIFY_AUTH_TOKEN")) != ""})
	case http.MethodPost:
		if !s.canWrite(w, r) {
			return
		}
		var payload deploymentPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		result, err := operations.Deploy(r.Context(), s.project, payload.Target, payload.Environment)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var input struct {
		Action  string `json:"action"`
		Starter string `json:"starter"`
	}
	if err := decodeJSON(r, &input); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	path := filepath.Join(s.project, ".mpress", "onboarding")
	if strings.EqualFold(strings.TrimSpace(input.Action), "starter") {
		if !s.onboarding() {
			writeAPIError(w, http.StatusConflict, errors.New("starter content can be changed only during first-run setup"))
			return
		}
		starter := strings.ToLower(strings.TrimSpace(input.Starter))
		switch starter {
		case "demo":
			// The generated tutorial is already present.
		case "empty":
			if err := s.replaceStarterWithMinimalSite(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, err)
				return
			}
			s.rebuild(true)
		case "import":
			if err := s.removeGeneratedStarter(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, err)
				return
			}
		default:
			writeAPIError(w, http.StatusBadRequest, errors.New("starter must be demo, empty, or import"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"starter": starter, "state": s.currentState()})
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"complete": true})
}

func (s *Server) replaceStarterWithMinimalSite() error {
	if err := s.removeGeneratedStarter(); err != nil {
		return err
	}
	content := s.cfg.ContentPath(s.project)
	index := "---\ntitle: Welcome\ndescription: Start writing your documentation.\n---\n\nReplace this page with your documentation.\n"
	navigation := "- label: Welcome\n  link: /\n"
	if err := os.WriteFile(filepath.Join(content, "index.md"), []byte(index), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(content, s.cfg.Build.NavFile), []byte(navigation), 0o644)
}

func (s *Server) removeGeneratedStarter() error {
	content := s.cfg.ContentPath(s.project)
	for _, name := range []string{"index.md", "getting-started.md", "components.md", s.cfg.Build.NavFile} {
		if err := os.Remove(filepath.Join(content, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, name := range []string{"component-light.svg", "component-dark.svg"} {
		if err := os.Remove(filepath.Join(s.cfg.StaticPath(s.project), "images", name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *Server) onboarding() bool {
	_, err := os.Stat(filepath.Join(s.project, ".mpress", "onboarding"))
	return err == nil
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	path, err := s.sourceForRoute(r.URL.Query().Get("url"))
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path})
}

// handleBrandAsset stores a theme asset in the project's static brand folder.
// The destination is selected by slot, not by the uploaded filename, so a
// browser cannot write outside the project's static directory.
func (s *Server) handleBrandAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	if err := r.ParseMultipartForm(maxBrandAssetSize + 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("choose an image file"))
		return
	}
	slot := strings.TrimSpace(r.FormValue("slot"))
	stems := map[string]string{
		"logoLight":   "logo-light",
		"logoDark":    "logo-dark",
		"favicon":     "favicon",
		"socialImage": "social-image",
	}
	stem, ok := stems[slot]
	if !ok {
		writeAPIError(w, http.StatusBadRequest, errors.New("unknown brand asset slot"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("choose an image file"))
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".svg": true, ".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".avif": true}
	if !allowed[ext] {
		writeAPIError(w, http.StatusBadRequest, errors.New("supported image formats are SVG, PNG, JPEG, WebP and AVIF"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBrandAssetSize+1))
	if err != nil || len(data) > maxBrandAssetSize {
		writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("image files must be 2 MB or smaller"))
		return
	}
	if len(data) == 0 {
		writeAPIError(w, http.StatusBadRequest, errors.New("image file is empty"))
		return
	}
	if ext == ".svg" {
		lower := bytes.ToLower(data)
		if !bytes.Contains(lower, []byte("<svg")) {
			writeAPIError(w, http.StatusBadRequest, errors.New("the SVG file is not valid"))
			return
		}
		if bytes.Contains(lower, []byte("<script")) || bytes.Contains(lower, []byte("javascript:")) || bytes.Contains(lower, []byte(" onload=")) {
			writeAPIError(w, http.StatusBadRequest, errors.New("SVG files must not contain scripts"))
			return
		}
	} else {
		detected := http.DetectContentType(data)
		valid := strings.HasPrefix(detected, "image/")
		if ext == ".avif" {
			valid = detected == "application/octet-stream" || detected == "image/avif"
		}
		if !valid {
			writeAPIError(w, http.StatusBadRequest, errors.New("the uploaded file is not a valid image"))
			return
		}
	}
	target := filepath.Join(s.project, s.cfg.Build.StaticDir, "brand", stem+ext)
	if err := s.writeSource(target, data, ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	rel := filepath.ToSlash(filepath.Join("brand", stem+ext))
	writeJSON(w, http.StatusOK, map[string]string{"slot": slot, "path": rel, "filename": filepath.Base(target)})
}

var imageAssetName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.(?:png|jpe?g|webp|avif)$`)

// handleImageAsset stores an image produced by the shared settings image
// editor. The category selects an allow-listed static folder. A caller must
// explicitly confirm an overwrite, and conflicts include a safe alternative.
func (s *Server) handleImageAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	if err := r.ParseMultipartForm(maxSourceSize + 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("choose an image file"))
		return
	}
	folders := map[string]string{
		"brand":   "brand",
		"blog":    filepath.Join("images", "blog"),
		"content": "images",
	}
	folder, ok := folders[strings.TrimSpace(r.FormValue("category"))]
	if !ok {
		writeAPIError(w, http.StatusBadRequest, errors.New("unknown image category"))
		return
	}
	name := filepath.Base(strings.TrimSpace(r.FormValue("filename")))
	if !imageAssetName.MatchString(name) {
		writeAPIError(w, http.StatusBadRequest, errors.New("use a PNG, JPEG, WebP, or AVIF filename with letters, numbers, dots, dashes, or underscores"))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("choose an image file"))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSourceSize+1))
	if err != nil || len(data) > maxSourceSize {
		writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("edited images must be 5 MB or smaller"))
		return
	}
	detected := http.DetectContentType(data)
	if !strings.HasPrefix(detected, "image/") {
		writeAPIError(w, http.StatusBadRequest, errors.New("the edited file is not a valid image"))
		return
	}
	target := filepath.Join(s.project, s.cfg.Build.StaticDir, folder, name)
	if _, err := os.Stat(target); err == nil && r.FormValue("overwrite") != "true" {
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		suggested := stem + "-edited" + ext
		for index := 2; ; index++ {
			if _, statErr := os.Stat(filepath.Join(filepath.Dir(target), suggested)); errors.Is(statErr, os.ErrNotExist) {
				break
			}
			suggested = fmt.Sprintf("%s-edited-%d%s", stem, index, ext)
		}
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "an image already uses this filename", "filename": name, "suggestedName": suggested,
		})
		return
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.writeSource(target, data, ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	rel := filepath.ToSlash(filepath.Join(folder, name))
	s.rebuild(true)
	writeJSON(w, http.StatusOK, map[string]string{"path": rel, "filename": name})
}

type blogPostPayload struct {
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Slug        string   `json:"slug"`
	Date        string   `json:"date"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Image       string   `json:"image"`
	Draft       bool     `json:"draft"`
	Body        string   `json:"body"`
	Revision    string   `json:"revision"`
}

type blogPostResponse struct {
	Path        string   `json:"path"`
	Route       string   `json:"route"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Slug        string   `json:"slug"`
	Date        string   `json:"date"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Image       string   `json:"image"`
	Draft       bool     `json:"draft"`
	Body        string   `json:"body,omitempty"`
	Revision    string   `json:"revision,omitempty"`
}

var blogPostSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func (s *Server) handleBlogPosts(w http.ResponseWriter, r *http.Request) {
	blogRoot := filepath.Join(s.cfg.ContentPath(s.project), "blog")
	switch r.Method {
	case http.MethodGet:
		if requested := strings.TrimSpace(r.URL.Query().Get("path")); requested != "" {
			target, err := s.sourcePath(requested, true)
			if err != nil || !pathWithin(blogRoot, target) || filepath.Base(target) == "index.md" {
				writeAPIError(w, http.StatusBadRequest, errors.New("choose a blog post inside the default blog directory"))
				return
			}
			data, err := os.ReadFile(target)
			if err != nil {
				writeAPIError(w, http.StatusNotFound, err)
				return
			}
			post, err := s.blogPostResponse(target, data, true)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, post)
			return
		}
		entries, err := os.ReadDir(blogRoot)
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusOK, map[string]any{"posts": []blogPostResponse{}})
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		posts := make([]blogPostResponse, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "index.md" {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".md" && ext != ".markdown" {
				continue
			}
			target := filepath.Join(blogRoot, entry.Name())
			data, readErr := os.ReadFile(target)
			if readErr != nil {
				continue
			}
			post, parseErr := s.blogPostResponse(target, data, false)
			if parseErr == nil {
				posts = append(posts, post)
			}
		}
		sort.SliceStable(posts, func(i, j int) bool {
			if posts[i].Date != posts[j].Date {
				return posts[i].Date > posts[j].Date
			}
			return posts[i].Title < posts[j].Title
		})
		writeJSON(w, http.StatusOK, map[string]any{"posts": posts})
	case http.MethodPost, http.MethodPut:
		if !s.canWrite(w, r) {
			return
		}
		var payload blogPostPayload
		if err := decodeJSON(r, &payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		payload.Title = strings.TrimSpace(payload.Title)
		payload.Slug = strings.TrimSpace(strings.ToLower(payload.Slug))
		payload.Date = strings.TrimSpace(payload.Date)
		if payload.Title == "" {
			writeAPIError(w, http.StatusBadRequest, errors.New("enter a post title"))
			return
		}
		if payload.Date != "" {
			if _, err := time.Parse("2006-01-02", payload.Date); err != nil {
				writeAPIError(w, http.StatusBadRequest, errors.New("publication date must use YYYY-MM-DD"))
				return
			}
		}
		if !payload.Draft && payload.Date == "" {
			payload.Date = time.Now().Format("2006-01-02")
		}
		var target string
		var current []byte
		var err error
		if r.Method == http.MethodPost {
			if !blogPostSlug.MatchString(payload.Slug) {
				writeAPIError(w, http.StatusBadRequest, errors.New("the post URL can contain lowercase letters, numbers and hyphens"))
				return
			}
			target = filepath.Join(blogRoot, payload.Slug+".md")
			if _, err = os.Stat(target); err == nil {
				writeAPIError(w, http.StatusConflict, errors.New("a blog post already uses this URL"))
				return
			}
		} else {
			target, err = s.sourcePath(payload.Path, true)
			if err != nil || !pathWithin(blogRoot, target) || filepath.Base(target) == "index.md" {
				writeAPIError(w, http.StatusBadRequest, errors.New("choose a blog post inside the default blog directory"))
				return
			}
			current, err = os.ReadFile(target)
			if err != nil {
				writeAPIError(w, http.StatusNotFound, err)
				return
			}
		}
		data, err := encodeBlogPost(current, payload)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		if err = s.writeSource(target, data, payload.Revision); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, errRevisionConflict) {
				status = http.StatusConflict
			}
			writeAPIError(w, status, err)
			return
		}
		s.rebuild(true)
		post, _ := s.blogPostResponse(target, data, true)
		status := http.StatusOK
		if r.Method == http.MethodPost {
			status = http.StatusCreated
		}
		writeJSON(w, status, map[string]any{"post": post, "state": s.currentState()})
	default:
		methodNotAllowed(w)
	}
}

func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (s *Server) blogPostResponse(target string, data []byte, includeBody bool) (blogPostResponse, error) {
	metadata, body, err := decodeBlogPost(data)
	if err != nil {
		return blogPostResponse{}, err
	}
	rel, _ := filepath.Rel(s.project, target)
	contentRel, _ := filepath.Rel(s.cfg.ContentPath(s.project), target)
	page, _, parseErr := content.NewRenderer().Parse(filepath.ToSlash(contentRel), s.cfg.Site.DefaultLanguage, string(data))
	if parseErr != nil {
		return blogPostResponse{}, parseErr
	}
	post := blogPostResponse{
		Path: filepath.ToSlash(rel), Route: "/" + strings.Trim(page.URLPath, "/") + "/",
		Title: stringValue(metadata["title"]), Description: stringValue(metadata["description"]),
		Slug: strings.TrimSuffix(filepath.Base(target), filepath.Ext(target)), Date: stringValue(metadata["date"]),
		Author: stringValue(metadata["author"]), Tags: stringSlice(metadata["tags"]), Image: stringValue(metadata["image"]),
		Draft: boolValue(metadata["draft"]), Revision: revision(data),
	}
	if includeBody {
		post.Body = strings.TrimSpace(body)
	}
	return post, nil
}

func decodeBlogPost(data []byte) (map[string]any, string, error) {
	lines := bytes.SplitAfter(data, []byte("\n"))
	if len(lines) < 2 || string(bytes.TrimSpace(lines[0])) != "---" {
		return nil, "", errors.New("the blog post requires YAML frontmatter")
	}
	closing := -1
	for index := 1; index < len(lines); index++ {
		if string(bytes.TrimSpace(lines[index])) == "---" {
			closing = index
			break
		}
	}
	if closing < 0 {
		return nil, "", errors.New("the blog post has unterminated YAML frontmatter")
	}
	metadata := map[string]any{}
	if err := yaml.Unmarshal(bytes.Join(lines[1:closing], nil), &metadata); err != nil {
		return nil, "", err
	}
	return metadata, string(bytes.Join(lines[closing+1:], nil)), nil
}

func encodeBlogPost(existing []byte, payload blogPostPayload) ([]byte, error) {
	metadata := map[string]any{}
	if len(existing) > 0 {
		var err error
		metadata, _, err = decodeBlogPost(existing)
		if err != nil {
			return nil, err
		}
	}
	metadata["title"] = payload.Title
	setOptionalMetadata(metadata, "description", strings.TrimSpace(payload.Description))
	setOptionalMetadata(metadata, "date", payload.Date)
	setOptionalMetadata(metadata, "author", strings.TrimSpace(payload.Author))
	setOptionalMetadata(metadata, "image", strings.TrimSpace(payload.Image))
	cleanTags := make([]string, 0, len(payload.Tags))
	for _, tag := range payload.Tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}
	if len(cleanTags) > 0 {
		metadata["tags"] = cleanTags
	} else {
		delete(metadata, "tags")
	}
	if payload.Draft {
		metadata["draft"] = true
	} else {
		delete(metadata, "draft")
	}
	header, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(payload.Body)
	return []byte("---\n" + string(header) + "---\n\n" + body + "\n"), nil
}

func setOptionalMetadata(metadata map[string]any, key, value string) {
	if value == "" {
		delete(metadata, key)
		return
	}
	metadata[key] = value
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if date, ok := value.(time.Time); ok {
		return date.Format("2006-01-02")
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func stringSlice(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		result := make([]string, 0, len(values))
		for _, item := range values {
			if text := stringValue(item); text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func (s *Server) handleMarkdownPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var payload struct {
		Markdown string `json:"markdown"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	page, diagnostics, err := content.NewRenderer().Parse(".mpress/blog-preview.md", s.cfg.Site.DefaultLanguage, payload.Markdown)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"html": page.HTML, "diagnostics": diagnostics})
}

type versionActionPayload struct {
	Action string `json:"action"`
	Label  string `json:"label"`
	Force  bool   `json:"force"`
}

func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request) {
	respond := func() {
		labels, err := docversion.List(s.project)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		type item struct {
			Label     string `json:"label"`
			CreatedAt string `json:"createdAt,omitempty"`
		}
		items := make([]item, 0, len(labels))
		for _, label := range labels {
			manifestPath := filepath.Join(s.cfg.ArtifactsPath(s.project), label, "mpress-version.json")
			var manifest docversion.Manifest
			if data, readErr := os.ReadFile(manifestPath); readErr == nil {
				_ = json.Unmarshal(data, &manifest)
			}
			items = append(items, item{Label: label, CreatedAt: manifest.CreatedAt})
		}
		writeJSON(w, http.StatusOK, map[string]any{"enabled": s.cfg.Version.Enabled, "current": s.cfg.Version.Current, "versions": items})
	}
	if r.Method == http.MethodGet {
		respond()
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var payload versionActionPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	payload.Label = strings.TrimSpace(payload.Label)
	switch payload.Action {
	case "capture":
		if payload.Label == "" {
			writeAPIError(w, http.StatusBadRequest, errors.New("enter a version label"))
			return
		}
		if err := s.captureVersion(payload); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	case "verify":
		if err := docversion.Verify(s.project, payload.Label); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	case "remove":
		if err := docversion.Remove(s.project, payload.Label); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		if s.cfg.Version.Current == payload.Label {
			updated := s.cfg
			updated.Version.Current = ""
			if err := s.saveConfiguration(updated); err != nil {
				writeAPIError(w, http.StatusBadRequest, err)
				return
			}
		}
		s.rebuild(true)
	case "current":
		if err := docversion.Verify(s.project, payload.Label); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		updated := s.cfg
		updated.Version.Enabled = true
		updated.Version.Current = payload.Label
		if err := s.saveConfiguration(updated); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		s.rebuild(true)
	default:
		writeAPIError(w, http.StatusBadRequest, errors.New("choose capture, verify, current or remove"))
		return
	}
	respond()
}

func (s *Server) captureVersion(payload versionActionPayload) error {
	// Keep the generated output stable while it is copied. The file watcher can
	// otherwise start another build after the configuration write and remove a
	// language directory while Capture is walking it.
	s.buildMu.Lock()
	defer s.buildMu.Unlock()

	configurationPath := filepath.Join(s.project, config.Filename)
	previousSource, err := os.ReadFile(configurationPath)
	if err != nil {
		return err
	}
	previousConfig := s.cfg
	updated := s.cfg
	updated.Version.Enabled = true
	if updated.Version.Current == "" {
		updated.Version.Current = payload.Label
	}
	if err := s.saveConfiguration(updated); err != nil {
		return err
	}
	restore := func(cause error) error {
		current, readErr := os.ReadFile(configurationPath)
		if readErr != nil {
			return fmt.Errorf("%v; could not restore configuration: %w", cause, readErr)
		}
		if restoreErr := s.writeSource(configurationPath, previousSource, revision(current)); restoreErr != nil {
			return fmt.Errorf("%v; could not restore configuration: %w", cause, restoreErr)
		}
		s.cfg = previousConfig
		_ = s.rebuildLocked(true)
		return cause
	}
	if err := s.rebuildLocked(false); err != nil {
		return restore(fmt.Errorf("build before version capture: %w", err))
	}
	if err := docversion.Capture(s.project, payload.Label, payload.Force); err != nil {
		return restore(err)
	}
	if err := s.rebuildLocked(true); err != nil {
		return fmt.Errorf("version %s was captured, but the site could not be rebuilt: %w", payload.Label, err)
	}
	return nil
}

func (s *Server) saveConfiguration(updated config.Config) error {
	if err := updated.Validate(); err != nil {
		return err
	}
	path := filepath.Join(s.project, config.Filename)
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(updated)
	if err != nil {
		return err
	}
	if err := s.writeSource(path, data, revision(current)); err != nil {
		return err
	}
	s.cfg = updated
	return nil
}

type blogImagePayload struct {
	Route       string  `json:"route"`
	Mode        string  `json:"mode,omitempty"`
	Fit         string  `json:"fit,omitempty"`
	Background  *string `json:"background,omitempty"`
	Width       *int    `json:"width,omitempty"`
	ShowTags    *bool   `json:"showTags,omitempty"`
	HeadingSize string  `json:"headingSize,omitempty"`
}

func (s *Server) handleBlogImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w)
		return
	}
	if !s.canWrite(w, r) {
		return
	}
	var payload blogImagePayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	payload.Mode = strings.ToLower(strings.TrimSpace(payload.Mode))
	payload.Fit = strings.ToLower(strings.TrimSpace(payload.Fit))
	payload.HeadingSize = strings.ToLower(strings.TrimSpace(payload.HeadingSize))
	if payload.Mode != "" && payload.Mode != "panel" && payload.Mode != "floating" {
		writeAPIError(w, http.StatusBadRequest, errors.New("image mode must be panel or floating"))
		return
	}
	if payload.Fit != "" && payload.Fit != "cover" && payload.Fit != "contain" {
		writeAPIError(w, http.StatusBadRequest, errors.New("image fit must be cover or contain"))
		return
	}
	if payload.Background != nil {
		background := strings.ToLower(strings.TrimSpace(*payload.Background))
		payload.Background = &background
		if background != "" && !validBlogImageBackground(background) {
			writeAPIError(w, http.StatusBadRequest, errors.New("image background must be a six-digit hex colour"))
			return
		}
	}
	if payload.HeadingSize != "" && payload.HeadingSize != "default" && payload.HeadingSize != "compact" && payload.HeadingSize != "large" {
		writeAPIError(w, http.StatusBadRequest, errors.New("heading size must be default, compact or large"))
		return
	}
	if payload.Width != nil && (*payload.Width < 50 || *payload.Width > 100) {
		writeAPIError(w, http.StatusBadRequest, errors.New("image width must be between 50 and 100 percent"))
		return
	}
	if payload.Mode == "floating" {
		background := ""
		payload.Background = &background
	}
	if payload.Mode == "" && payload.Fit == "" && payload.Background == nil && payload.Width == nil && payload.ShowTags == nil && payload.HeadingSize == "" {
		writeAPIError(w, http.StatusBadRequest, errors.New("choose a blog presentation setting"))
		return
	}
	rel, err := s.sourceForRoute(payload.Route)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	path, err := s.sourcePath(rel, true)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	current, err := os.ReadFile(path)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	updated := current
	if payload.Mode != "" {
		if payload.Mode == "panel" {
			updated, err = removeMarkdownFrontmatterValue(updated, "imageMode")
		} else {
			updated, err = setMarkdownFrontmatterValue(updated, "imageMode", payload.Mode)
		}
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	}
	if payload.Fit != "" {
		updated, err = setMarkdownFrontmatterValue(updated, "imageFit", payload.Fit)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	}
	if payload.Background != nil {
		if *payload.Background == "" {
			updated, err = removeMarkdownFrontmatterValue(updated, "imageBackground")
		} else {
			updated, err = setMarkdownFrontmatterValue(updated, "imageBackground", *payload.Background)
		}
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	}
	if payload.Width != nil {
		if *payload.Width == 100 {
			updated, err = removeMarkdownFrontmatterValue(updated, "imageWidth")
		} else {
			updated, err = setMarkdownFrontmatterValue(updated, "imageWidth", *payload.Width)
		}
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	}
	if payload.ShowTags != nil {
		if *payload.ShowTags {
			updated, err = removeMarkdownFrontmatterValue(updated, "showTags")
		} else {
			updated, err = setMarkdownFrontmatterValue(updated, "showTags", false)
		}
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	}
	if payload.HeadingSize != "" {
		if payload.HeadingSize == "default" {
			updated, err = removeMarkdownFrontmatterValue(updated, "headingSize")
		} else {
			updated, err = setMarkdownFrontmatterValue(updated, "headingSize", payload.HeadingSize)
		}
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
	}
	if err := s.writeSource(path, updated, revision(current)); err != nil {
		writeAPIError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": payload.Mode, "fit": payload.Fit, "background": payload.Background, "width": payload.Width, "showTags": payload.ShowTags, "headingSize": payload.HeadingSize, "path": rel, "state": s.currentState()})
}

func validBlogImageBackground(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, character := range value[1:] {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func (s *Server) sourceForRoute(rawURL string) (string, error) {
	data, err := os.ReadFile(filepath.Join(s.cfg.OutputPath(s.project), "mpress-manifest.json"))
	if err != nil {
		return "", err
	}
	var manifest struct {
		Pages []struct{ Source, URL, Language string } `json:"pages"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", err
	}
	wanted := strings.Trim(rawURL, "/")
	for _, page := range manifest.Pages {
		route := strings.Trim(page.URL, "/")
		if page.Language != s.cfg.Site.DefaultLanguage || !s.cfg.Site.DefaultAtRoot {
			route = strings.Trim(page.Language+"/"+route, "/")
		}
		if route == wanted {
			source := page.Source
			if page.Language != s.cfg.Site.DefaultLanguage {
				source = filepath.ToSlash(filepath.Join(page.Language, source))
			}
			return filepath.ToSlash(filepath.Join(s.cfg.Build.ContentDir, source)), nil
		}
	}
	return "", errors.New("the current route does not map to a Markdown source file")
}

func (s *Server) handleSite(w http.ResponseWriter, r *http.Request) {
	recorder := httptest.NewRecorder()
	s.fileServer.ServeHTTP(recorder, r)
	for key, values := range recorder.Header() {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	body := recorder.Body.Bytes()
	if r.Method == http.MethodGet && recorder.Code == http.StatusOK && strings.HasPrefix(recorder.Header().Get("Content-Type"), "text/html") && r.Header.Get("X-MPress-Audit") != "lighthouse" && r.URL.Query().Get("__mpress_preview") != "1" {
		workspace := r.Header.Get("X-MPress-Workspace") == "settings"
		hostAttribute := ""
		if workspace {
			hostAttribute = " data-mpress-workspace"
			baseHref := r.URL.Path
			if !strings.HasSuffix(baseHref, "/") {
				baseHref = path.Dir(baseHref) + "/"
			}
			// The generated pages use route-relative asset URLs. Put the base element
			// at the start of head, before the stylesheet is parsed, so the editor
			// route does not accidentally request /__mpress/assets/… instead.
			body = injectBaseHref(body, baseHref)
		}
		injection := []byte(`<style id="mpress-devbar-page-style">
:root{--mpress-devbar-height:42px;scroll-padding-bottom:58px}
body{padding-bottom:42px!important}
.mpress-responsive-preview-stage{position:fixed;z-index:2147482000;inset:0 0 42px;display:grid;justify-content:center;overflow:hidden;background:color-mix(in srgb,var(--bg,#0b111b) 90%,#000);}
.mpress-responsive-preview-frame{display:block;width:var(--mpress-preview-width);max-width:100vw;height:100%;border:0;border-inline:1px solid color-mix(in srgb,var(--border,#2a3749) 82%,transparent);background:var(--bg,#fff);box-shadow:0 0 60px #0008;}
[data-blog-style-editor]{position:relative}
.sidebar,.blog-sidebar{padding-bottom:72px!important}
.mpress-dev-edit-site{position:absolute;right:12px;bottom:12px;left:12px;z-index:4;display:flex;min-height:42px;align-items:center;gap:9px;padding:0 12px;border:1px solid var(--border);border-radius:6px;background:color-mix(in srgb,var(--surface-solid) 94%,transparent);color:var(--text);font:650 13px/1.2 ui-sans-serif,system-ui,sans-serif;text-decoration:none;box-shadow:0 10px 28px color-mix(in srgb,var(--bg) 52%,transparent);backdrop-filter:blur(14px)}
.mpress-dev-edit-site:hover{border-color:color-mix(in srgb,var(--accent) 42%,var(--border));background:var(--panel);color:var(--text);text-decoration:none}
.mpress-dev-edit-site:focus-visible{outline:2px solid var(--accent);outline-offset:2px}
.mpress-dev-edit-site .lucide{width:16px;height:16px;color:var(--accent)}
.mpress-workspace-document .mpress-workspace-sidebar nav{padding-bottom:82px}
.mpress-workspace-document .mpress-workspace-sidebar nav>*+*{margin-top:1px!important}
.mpress-workspace-document .mpress-workspace-sidebar .nav-label{margin-top:18px!important;padding-bottom:5px;color:var(--muted);font-size:11px;font-weight:750;letter-spacing:.09em;text-transform:uppercase}
.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-tools-label{margin-top:24px!important}
.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link{display:flex;min-height:34px;align-items:center;gap:10px;padding:.38rem .55rem;font-size:14px;font-weight:400}
.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link>.lucide{width:16px;height:16px;flex:0 0 16px;color:color-mix(in srgb,var(--muted) 82%,transparent)}
.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link:hover>.lucide,.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link.active>.lucide{color:currentColor}
.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-back{margin-bottom:15px!important;color:var(--text);font-weight:600}
.mpress-dev-image-edit{position:absolute;z-index:5;top:12px;right:12px;display:grid;width:36px;height:36px;padding:0;place-items:center;border:1px solid #38465a;border-radius:7px;background:#0c131fee;color:#eef3fa;box-shadow:0 8px 24px #03070c66;cursor:pointer;backdrop-filter:blur(12px)}
.mpress-dev-image-edit:hover,.mpress-dev-image-edit[aria-expanded="true"]{border-color:#7593ff;background:#172231;color:#fff}
.mpress-dev-image-edit .lucide{width:16px;height:16px}
.mpress-dev-image-resize{position:absolute;z-index:4;display:grid;width:24px;height:44px;padding:0;place-items:center;border:0;border-radius:999px;outline:0;background:transparent;color:#a9b5c6;cursor:ew-resize;touch-action:none;transform:translate(-50%,-50%)}
.mpress-dev-image-resize::before{content:"";position:absolute;inset:7px 8px;border:1px solid #70809766;border-radius:999px;background:#0c131f80;box-shadow:0 3px 10px #03070c30;backdrop-filter:blur(6px);transition:border-color .15s ease,background .15s ease,box-shadow .15s ease}
.mpress-dev-image-resize[hidden]{display:none}
.mpress-dev-image-resize:hover,.mpress-dev-image-resize:focus-visible,.mpress-dev-image-resize.active{color:#eef3fa}
.mpress-dev-image-resize:hover::before,.mpress-dev-image-resize:focus-visible::before,.mpress-dev-image-resize.active::before{border-color:#91a4c080;background:#172231cc;box-shadow:0 0 0 2px #8ba1c51f,0 4px 12px #03070c3d}
.mpress-dev-image-resize:disabled{cursor:wait;opacity:.7}
.mpress-dev-image-resize .lucide{position:relative;width:10px;height:10px;pointer-events:none}
.mpress-dev-image-menu{position:absolute;z-index:5;top:56px;right:12px;display:grid;grid-template-columns:1fr 1fr;gap:5px;width:244px;padding:8px;border:1px solid #38465a;border-radius:8px;background:#0c131ff5;color:#eef3fa;box-shadow:0 18px 46px #03070c88;font:13px/1.35 ui-sans-serif,system-ui,sans-serif;backdrop-filter:blur(16px)}
.mpress-dev-image-menu[hidden]{display:none}
.mpress-dev-image-section{grid-column:1/-1;padding:3px 5px 6px;color:#9aa8bb;font-size:10px;font-weight:800;letter-spacing:.1em;text-transform:uppercase}
.mpress-dev-image-menu button{min-height:34px;padding:0 9px;border:1px solid #2a3749;border-radius:5px;background:#111a27;color:#9aa8bb;cursor:pointer;font:inherit;font-weight:650}
.mpress-dev-image-menu button:hover{background:#172231;color:#eef3fa}
.mpress-dev-image-menu button.active{border-color:#5278ff;background:#203460;color:#fff}
.mpress-dev-image-controls{grid-column:1/-1;display:grid;grid-template-columns:1fr 1fr;gap:5px}
.mpress-dev-image-panel-controls{grid-column:1/-1;display:grid;grid-template-columns:1fr 1fr;gap:5px}
.mpress-dev-image-panel-controls[hidden]{display:none}
.mpress-dev-image-background{grid-column:1/-1;display:flex;align-items:center;justify-content:space-between;gap:10px;margin-top:5px;padding:8px 5px 3px;border-top:1px solid #2a3749;color:#9aa8bb;font-size:11px;font-weight:700}
.mpress-dev-image-background input{width:42px;height:30px;padding:2px;border:1px solid #38465a;border-radius:5px;background:#111a27;cursor:pointer}
.mpress-dev-image-reset{grid-column:1/-1}
.mpress-dev-image-heading-options{grid-column:1/-1;display:grid;grid-template-columns:repeat(3,1fr);gap:5px}
</style><div id="mpress-devbar-host"` + hostAttribute + `></div><script src="/__mpress/devbar.js" defer></script>`)
		if index := bytes.LastIndex(body, []byte("</body>")); index >= 0 {
			body = append(append(append([]byte{}, body[:index]...), injection...), body[index:]...)
		} else {
			body = append(body, injection...)
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	}
	w.WriteHeader(recorder.Code)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func injectBaseHref(body []byte, href string) []byte {
	base := []byte(`<base href="` + html.EscapeString(href) + `"><style id="mpress-workspace-boot-style">body:not(.mpress-workspace-document)>:not(header):not(script):not(style):not(#mpress-devbar-host){visibility:hidden!important}</style>`)
	lower := bytes.ToLower(body)
	if start := bytes.Index(lower, []byte("<head")); start >= 0 {
		if end := bytes.IndexByte(lower[start:], '>'); end >= 0 {
			index := start + end + 1
			return append(append(append([]byte{}, body[:index]...), base...), body[index:]...)
		}
	}
	if index := bytes.Index(lower, []byte("</head>")); index >= 0 {
		return append(append(append([]byte{}, body[:index]...), base...), body[index:]...)
	}
	return body
}

func setYAMLValue(data []byte, path []string, value any) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if len(document.Content) == 0 {
		document.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	node := document.Content[0]
	for index, key := range path {
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s is not a configuration map", strings.Join(path[:index], "."))
		}
		var next *yaml.Node
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				next = node.Content[i+1]
				break
			}
		}
		if index == len(path)-1 {
			var replacement yaml.Node
			if err := replacement.Encode(value); err != nil {
				return nil, err
			}
			if next == nil {
				node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &replacement)
			} else {
				commentHead, commentLine, commentFoot := next.HeadComment, next.LineComment, next.FootComment
				*next = replacement
				next.HeadComment, next.LineComment, next.FootComment = commentHead, commentLine, commentFoot
			}
			break
		}
		if next == nil {
			next = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, next)
		}
		node = next
	}
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, err
	}
	return output.Bytes(), encoder.Close()
}

func setMarkdownFrontmatterValue(data []byte, key string, value any) ([]byte, error) {
	return updateMarkdownFrontmatter(data, func(header []byte) ([]byte, error) {
		return setYAMLValue(header, []string{key}, value)
	})
}

func removeMarkdownFrontmatterValue(data []byte, key string) ([]byte, error) {
	return updateMarkdownFrontmatter(data, func(header []byte) ([]byte, error) {
		var document yaml.Node
		if err := yaml.Unmarshal(header, &document); err != nil {
			return nil, err
		}
		if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
			return header, nil
		}
		mapping := document.Content[0]
		for index := 0; index+1 < len(mapping.Content); index += 2 {
			if mapping.Content[index].Value == key {
				mapping.Content = append(mapping.Content[:index], mapping.Content[index+2:]...)
				break
			}
		}
		var output bytes.Buffer
		encoder := yaml.NewEncoder(&output)
		encoder.SetIndent(2)
		if err := encoder.Encode(&document); err != nil {
			return nil, err
		}
		return output.Bytes(), encoder.Close()
	})
}

func updateMarkdownFrontmatter(data []byte, update func([]byte) ([]byte, error)) ([]byte, error) {
	lines := bytes.SplitAfter(data, []byte("\n"))
	if len(lines) < 2 || string(bytes.TrimSpace(lines[0])) != "---" {
		return nil, errors.New("the blog post requires YAML frontmatter")
	}
	closing := -1
	for index := 1; index < len(lines); index++ {
		if string(bytes.TrimSpace(lines[index])) == "---" {
			closing = index
			break
		}
	}
	if closing < 0 {
		return nil, errors.New("the blog post has unterminated YAML frontmatter")
	}
	header := bytes.Join(lines[1:closing], nil)
	updatedHeader, err := update(header)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	output.Write(lines[0])
	output.Write(updatedHeader)
	output.Write(bytes.Join(lines[closing:], nil))
	return output.Bytes(), nil
}

func (s *Server) canWrite(w http.ResponseWriter, r *http.Request) bool {
	if !s.options.Authoring {
		writeAPIError(w, http.StatusForbidden, errors.New("authoring is disabled; restart with --authoring"))
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		if !strings.EqualFold(strings.TrimSuffix(origin, "/"), "http://"+r.Host) && !strings.EqualFold(strings.TrimSuffix(origin, "/"), "https://"+r.Host) {
			writeAPIError(w, http.StatusForbidden, errors.New("cross-origin writes are not allowed"))
			return false
		}
	}
	if s.requireWriteToken && r.Header.Get("X-MPress-Token") != s.token {
		writeAPIError(w, http.StatusUnauthorized, errors.New("a valid authoring token is required"))
		return false
	}
	return true
}

func (s *Server) sourcePath(name string, mutation bool) (string, error) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	if name == "" || strings.ContainsRune(name, 0) || filepath.IsAbs(name) {
		return "", errors.New("invalid source path")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("source path must stay inside the project")
	}
	allowed := clean == config.Filename || clean == ".gitignore" || clean == s.cfg.Build.NavFile ||
		clean == s.cfg.Build.ContentDir || strings.HasPrefix(clean, strings.TrimSuffix(s.cfg.Build.ContentDir, "/")+"/") ||
		clean == s.cfg.Build.StaticDir || strings.HasPrefix(clean, strings.TrimSuffix(s.cfg.Build.StaticDir, "/")+"/")
	if !allowed {
		return "", errors.New("source path is not editable")
	}
	if mutation && !editableExtension(clean) {
		return "", errors.New("this file type is not editable through the authoring API")
	}
	path := filepath.Join(s.project, filepath.FromSlash(clean))
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(s.project, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("source path must stay inside the project")
	}
	return abs, nil
}

func editableExtension(path string) bool {
	if path == config.Filename || path == ".gitignore" {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".yaml", ".yml", ".css", ".js", ".json", ".txt":
		return true
	default:
		return false
	}
}

var errRevisionConflict = errors.New("the file changed since it was opened; reload it before saving")

func (s *Server) writeSource(path string, data []byte, expected string) error {
	current, err := os.ReadFile(path)
	if err == nil {
		if expected != "" && revision(current) != expected {
			return errRevisionConflict
		}
		if err := s.backup(path, current); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mpress-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (s *Server) backup(path string, data []byte) error {
	rel, err := filepath.Rel(s.project, path)
	if err != nil {
		return err
	}
	dir := time.Now().UTC().Format("20060102T150405.000000000Z")
	target := filepath.Join(s.project, ".mpress", "backups", dir, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func revision(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}

func newToken() (string, error) {
	data := make([]byte, 18)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (s *Server) routeFor(projectPath, source string) string {
	contentPrefix := strings.TrimSuffix(filepath.ToSlash(s.cfg.Build.ContentDir), "/") + "/"
	rel := strings.TrimPrefix(filepath.ToSlash(projectPath), contentPrefix)
	page, _, err := content.NewRenderer().Parse(rel, s.cfg.Site.DefaultLanguage, source)
	if err != nil {
		return ""
	}
	if page.URLPath == "" {
		return "/"
	}
	return "/" + page.URLPath + "/"
}

func (s *Server) rebuild(reload bool) {
	s.buildMu.Lock()
	defer s.buildMu.Unlock()
	_ = s.rebuildLocked(reload)
}

func (s *Server) rebuildLocked(reload bool) error {
	s.setState(BuildState{Status: "building", Revision: s.currentState().Revision + 1})
	result, err := site.Build(s.project, site.BuildOptions{IncludeDrafts: true})
	state := BuildState{Status: "ready", Pages: result.Pages, Files: result.Files, DurationMS: result.Duration.Milliseconds(), Diagnostics: result.Diagnostics, BuiltAt: time.Now(), Revision: s.currentState().Revision}
	if err != nil {
		state.Status = "error"
		state.Error = err.Error()
	}
	s.setState(state)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rebuild:", err)
	} else if reload {
		fmt.Printf("Rebuilt %d pages in %dms\n", state.Pages, state.DurationMS)
		s.events.send("reload", state)
	}
	return err
}

func (s *Server) setState(state BuildState) {
	s.stateMu.Lock()
	s.state = state
	s.stateMu.Unlock()
	s.events.send("state", state)
}

func (s *Server) currentState() BuildState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

func (s *Server) watch() {
	last := latest(s.project, s.cfg.OutputPath(s.project))
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		next := latest(s.project, s.cfg.OutputPath(s.project))
		if next.After(last) {
			last = next
			s.rebuild(true)
		}
	}
}

func latest(root, output string) time.Time {
	var latest time.Time
	absOutput, _ := filepath.Abs(output)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		absPath, _ := filepath.Abs(path)
		if entry.IsDir() {
			if path != root && (absPath == absOutput || entry.Name() == ".git" || entry.Name() == ".mpress") {
				return fs.SkipDir
			}
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest
}

type hub struct {
	mu      sync.Mutex
	clients map[chan event]struct{}
}

type event struct {
	name string
	data any
}

func newHub() *hub { return &hub{clients: map[chan event]struct{}{}} }

func (h *hub) handle(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	ch := make(chan event, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	_, _ = io.WriteString(w, "event: ready\ndata: {}\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case message := <-ch:
			data, _ := json.Marshal(message.data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", message.name, data)
			flusher.Flush()
		}
	}
}

func (h *hub) send(name string, data any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- event{name: name, data: data}:
		default:
		}
	}
}

func decodeJSON(r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxSourceSize+1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeAPIError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}
