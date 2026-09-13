package dev

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/operations"
	docversion "github.com/leaanthony/mpress/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

type mcpEmptyInput struct{}

type mcpBuildSummary struct {
	Status     string `json:"status"`
	Pages      int    `json:"pages"`
	Files      int    `json:"files"`
	DurationMS int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
	Revision   int64  `json:"revision"`
}

type mcpProjectOutput struct {
	Name      string          `json:"name"`
	Title     string          `json:"title"`
	Root      string          `json:"root"`
	Authoring bool            `json:"authoring"`
	Build     mcpBuildSummary `json:"build"`
}

type mcpFileEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type mcpListFilesOutput struct {
	Files []mcpFileEntry `json:"files"`
}

type mcpReadFileInput struct {
	Path string `json:"path" jsonschema:"Project-relative path to an M-Press source file."`
}

type mcpFileOutput struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Revision string `json:"revision"`
	Route    string `json:"route,omitempty"`
}

type mcpWriteFileInput struct {
	Path     string `json:"path" jsonschema:"Project-relative path to an editable source file."`
	Content  string `json:"content" jsonschema:"Complete replacement content for the file."`
	Revision string `json:"revision" jsonschema:"Revision returned by read_file. Use an empty string only when creating a new file."`
}

type mcpWriteFileOutput struct {
	File  mcpFileOutput   `json:"file"`
	Build mcpBuildSummary `json:"build"`
}

type mcpGetConfigOutput struct {
	Config   config.Config `json:"config"`
	Revision string        `json:"revision"`
}

type mcpUpdateConfigInput struct {
	Config   config.Config `json:"config" jsonschema:"The complete M-Press configuration object."`
	Revision string        `json:"revision" jsonschema:"Revision returned by get_config."`
}

type mcpUpdateConfigOutput struct {
	Revision string          `json:"revision"`
	Build    mcpBuildSummary `json:"build"`
}

type mcpDiagnostic struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Message  string `json:"message"`
}

type mcpBrokenLink struct {
	File   string `json:"file"`
	Href   string `json:"href"`
	Reason string `json:"reason"`
}

type mcpCheckOutput struct {
	OK          bool            `json:"ok"`
	Pages       int             `json:"pages"`
	Files       int             `json:"files"`
	Diagnostics []mcpDiagnostic `json:"diagnostics"`
	Broken      []mcpBrokenLink `json:"broken"`
}

type mcpCaptureVersionInput struct {
	Label string `json:"label" jsonschema:"Immutable version label, for example 1.0 or 2026-08."`
	Force bool   `json:"force,omitempty" jsonschema:"Replace an existing captured version with the same label."`
}

type mcpVersionsOutput struct {
	Versions []string `json:"versions"`
}

type mcpDeployInput struct {
	Target      string `json:"target,omitempty" jsonschema:"Configured deployment target. Leave empty to use the default target."`
	Environment string `json:"environment,omitempty" jsonschema:"Deployment environment: preview or production. Defaults to preview."`
}

type mcpDeployOutput struct {
	Provider     string `json:"provider"`
	Target       string `json:"target"`
	Environment  string `json:"environment"`
	DeploymentID string `json:"deploymentId"`
	URL          string `json:"url"`
	Uploaded     int    `json:"uploaded"`
	Reused       int    `json:"reused"`
}

func (s *Server) newMCPHandler() http.Handler {
	serverVersion := strings.TrimSpace(s.options.Version)
	if serverVersion == "" {
		serverVersion = "0.1.0-dev"
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "mpress", Title: "M-Press", Version: serverVersion}, &mcp.ServerOptions{
		Instructions: "Use M-Press tools to inspect and maintain the current documentation project. Read a file or configuration before changing it, then send the returned revision with the write.",
	})

	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPointer(false)}
	localWrite := &mcp.ToolAnnotations{DestructiveHint: boolPointer(true), IdempotentHint: true, OpenWorldHint: boolPointer(false)}

	mcp.AddTool(server, &mcp.Tool{Name: "project_info", Title: "Get M-Press project information", Description: "Return the current project identity and build state.", Annotations: readOnly},
		func(_ context.Context, _ *mcp.CallToolRequest, _ mcpEmptyInput) (*mcp.CallToolResult, mcpProjectOutput, error) {
			return nil, mcpProjectOutput{Name: filepath.Base(s.project), Title: s.cfg.Site.Title, Root: s.project, Authoring: s.options.Authoring, Build: mcpBuild(s.currentState())}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "list_files", Title: "List M-Press source files", Description: "List editable project files and directories.", Annotations: readOnly},
		func(_ context.Context, _ *mcp.CallToolRequest, _ mcpEmptyInput) (*mcp.CallToolResult, mcpListFilesOutput, error) {
			var files []mcpFileEntry
			for _, name := range []string{s.cfg.Build.ContentDir, s.cfg.Build.StaticDir, config.Filename} {
				path := filepath.Join(s.project, filepath.FromSlash(name))
				if _, err := os.Stat(path); err != nil {
					continue
				}
				node, err := s.readTree(path)
				if err != nil {
					return nil, mcpListFilesOutput{}, err
				}
				appendMCPFiles(node, &files)
			}
			return nil, mcpListFilesOutput{Files: files}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "read_file", Title: "Read an M-Press source file", Description: "Read a source file with the revision required for a safe write.", Annotations: readOnly},
		func(_ context.Context, _ *mcp.CallToolRequest, input mcpReadFileInput) (*mcp.CallToolResult, mcpFileOutput, error) {
			file, err := s.mcpReadFile(input.Path)
			return nil, file, err
		})

	mcp.AddTool(server, &mcp.Tool{Name: "write_file", Title: "Write an M-Press source file", Description: "Safely replace an editable source file, create a backup, rebuild the site, and trigger live reload.", Annotations: localWrite},
		func(_ context.Context, _ *mcp.CallToolRequest, input mcpWriteFileInput) (*mcp.CallToolResult, mcpWriteFileOutput, error) {
			if err := s.requireMCPAuthoring(); err != nil {
				return nil, mcpWriteFileOutput{}, err
			}
			path, err := s.sourcePath(input.Path, true)
			if err != nil {
				return nil, mcpWriteFileOutput{}, err
			}
			if len(input.Content) > maxSourceSize {
				return nil, mcpWriteFileOutput{}, errors.New("file is too large to edit")
			}
			if _, err := os.Stat(path); err == nil && strings.TrimSpace(input.Revision) == "" {
				return nil, mcpWriteFileOutput{}, errors.New("revision is required when replacing an existing file")
			}
			if err := s.writeSource(path, []byte(input.Content), input.Revision); err != nil {
				return nil, mcpWriteFileOutput{}, err
			}
			s.rebuild(true)
			file, err := s.mcpReadFile(input.Path)
			return nil, mcpWriteFileOutput{File: file, Build: mcpBuild(s.currentState())}, err
		})

	mcp.AddTool(server, &mcp.Tool{Name: "get_config", Title: "Get the complete M-Press configuration", Description: "Return every supported M-Press configuration field with a safe-write revision.", Annotations: readOnly},
		func(_ context.Context, _ *mcp.CallToolRequest, _ mcpEmptyInput) (*mcp.CallToolResult, mcpGetConfigOutput, error) {
			data, err := os.ReadFile(filepath.Join(s.project, config.Filename))
			return nil, mcpGetConfigOutput{Config: s.cfg, Revision: revision(data)}, err
		})

	mcp.AddTool(server, &mcp.Tool{Name: "update_config", Title: "Update the complete M-Press configuration", Description: "Validate and replace the complete M-Press configuration, then rebuild the site.", Annotations: localWrite},
		func(_ context.Context, _ *mcp.CallToolRequest, input mcpUpdateConfigInput) (*mcp.CallToolResult, mcpUpdateConfigOutput, error) {
			if err := s.requireMCPAuthoring(); err != nil {
				return nil, mcpUpdateConfigOutput{}, err
			}
			if strings.TrimSpace(input.Revision) == "" {
				return nil, mcpUpdateConfigOutput{}, errors.New("revision is required when updating the configuration")
			}
			updated := input.Config
			if err := updated.Validate(); err != nil {
				return nil, mcpUpdateConfigOutput{}, err
			}
			data, err := yaml.Marshal(updated)
			if err != nil {
				return nil, mcpUpdateConfigOutput{}, err
			}
			path := filepath.Join(s.project, config.Filename)
			if err := s.writeSource(path, data, input.Revision); err != nil {
				return nil, mcpUpdateConfigOutput{}, err
			}
			s.cfg = updated
			s.rebuild(true)
			return nil, mcpUpdateConfigOutput{Revision: revision(data), Build: mcpBuild(s.currentState())}, nil
		})

	mcp.AddTool(server, &mcp.Tool{Name: "check_site", Title: "Check the M-Press site", Description: "Run a strict build and verify every internal link and asset.", Annotations: readOnly},
		func(_ context.Context, _ *mcp.CallToolRequest, _ mcpEmptyInput) (*mcp.CallToolResult, mcpCheckOutput, error) {
			result, checkErr := operations.Check(s.project, true)
			output := mcpCheckOutput{OK: checkErr == nil, Pages: result.Build.Pages, Files: result.Build.Files}
			for _, diagnostic := range result.Build.Diagnostics {
				output.Diagnostics = append(output.Diagnostics, mcpDiagnostic{Severity: diagnostic.Severity, File: diagnostic.File, Message: diagnostic.Message})
			}
			for _, broken := range result.Broken {
				output.Broken = append(output.Broken, mcpBrokenLink{File: broken.File, Href: broken.Href, Reason: broken.Reason})
			}
			return nil, output, checkErr
		})

	mcp.AddTool(server, &mcp.Tool{Name: "list_versions", Title: "List captured documentation versions", Description: "List immutable documentation versions captured by M-Press.", Annotations: readOnly},
		func(_ context.Context, _ *mcp.CallToolRequest, _ mcpEmptyInput) (*mcp.CallToolResult, mcpVersionsOutput, error) {
			versions, err := docversion.List(s.project)
			return nil, mcpVersionsOutput{Versions: versions}, err
		})

	mcp.AddTool(server, &mcp.Tool{Name: "capture_version", Title: "Capture a documentation version", Description: "Capture the current generated site as an immutable documentation version.", Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPointer(false), OpenWorldHint: boolPointer(false)}},
		func(_ context.Context, _ *mcp.CallToolRequest, input mcpCaptureVersionInput) (*mcp.CallToolResult, mcpVersionsOutput, error) {
			if err := s.requireMCPAuthoring(); err != nil {
				return nil, mcpVersionsOutput{}, err
			}
			if _, err := operations.Check(s.project, false); err != nil {
				return nil, mcpVersionsOutput{}, err
			}
			if err := docversion.Capture(s.project, input.Label, input.Force); err != nil {
				return nil, mcpVersionsOutput{}, err
			}
			versions, err := docversion.List(s.project)
			return nil, mcpVersionsOutput{Versions: versions}, err
		})

	mcp.AddTool(server, &mcp.Tool{Name: "deploy_site", Title: "Deploy the M-Press site", Description: "Run deployment checks and publish a preview or production build to a configured target.", Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPointer(true), OpenWorldHint: boolPointer(true)}},
		func(ctx context.Context, _ *mcp.CallToolRequest, input mcpDeployInput) (*mcp.CallToolResult, mcpDeployOutput, error) {
			if err := s.requireMCPAuthoring(); err != nil {
				return nil, mcpDeployOutput{}, err
			}
			result, err := operations.Deploy(ctx, s.project, input.Target, input.Environment)
			return nil, mcpDeployOutput{Provider: result.Provider, Target: result.Target, Environment: result.Environment, DeploymentID: result.Deployment, URL: result.URL, Uploaded: result.Uploaded, Reused: result.Reused}, err
		})

	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}

func (s *Server) authoriseMCP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := strings.TrimSuffix(strings.TrimSpace(r.Header.Get("Origin")), "/"); origin != "" && !strings.EqualFold(origin, "http://"+r.Host) && !strings.EqualFold(origin, "https://"+r.Host) {
			http.Error(w, "cross-origin MCP requests are not allowed", http.StatusForbidden)
			return
		}
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if len(authorization) < 8 || !strings.EqualFold(authorization[:7], "Bearer ") || !secureTokenEqual(strings.TrimSpace(authorization[7:]), s.token) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mpress-mcp"`)
			http.Error(w, "a valid M-Press server token is required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) mcpReadFile(name string) (mcpFileOutput, error) {
	path, err := s.sourcePath(name, false)
	if err != nil {
		return mcpFileOutput{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return mcpFileOutput{}, err
	}
	if len(data) > maxSourceSize {
		return mcpFileOutput{}, errors.New("file is too large to edit")
	}
	rel, _ := filepath.Rel(s.project, path)
	projectPath := filepath.ToSlash(rel)
	return mcpFileOutput{Path: projectPath, Content: string(data), Revision: revision(data), Route: s.routeFor(projectPath, string(data))}, nil
}

func (s *Server) requireMCPAuthoring() error {
	if !s.options.Authoring {
		return errors.New("authoring is disabled; restart M-Press without --read-only")
	}
	return nil
}

func appendMCPFiles(node treeNode, files *[]mcpFileEntry) {
	*files = append(*files, mcpFileEntry{Path: node.Path, Kind: node.Kind})
	for _, child := range node.Children {
		appendMCPFiles(child, files)
	}
}

func mcpBuild(state BuildState) mcpBuildSummary {
	return mcpBuildSummary{Status: state.Status, Pages: state.Pages, Files: state.Files, DurationMS: state.DurationMS, Error: state.Error, Revision: state.Revision}
}

func boolPointer(value bool) *bool { return &value }

func secureTokenEqual(left, right string) bool {
	return left != "" && right != "" && len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
