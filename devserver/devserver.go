// Package devserver exposes the M-Press development server to optional local
// tools without making the parser, renderer, or file-safety internals public.
package devserver

import "github.com/leaanthony/mpress/internal/dev"

type Options struct {
	Host      string
	Port      int
	Authoring bool
	Token     string
	Workspace *WorkspaceAssets
}

// WorkspaceAssets is an optional browser interface supplied by another product.
// The open-source M-Press binary does not embed a browser editor bundle.
type WorkspaceAssets struct {
	HTML string
	CSS  string
	JS   string
}

func Serve(project string, options Options) error {
	var workspace *dev.WorkspaceAssets
	if options.Workspace != nil {
		workspace = &dev.WorkspaceAssets{HTML: options.Workspace.HTML, CSS: options.Workspace.CSS, JS: options.Workspace.JS}
	}
	return dev.Serve(project, dev.Options{
		Host: options.Host, Port: options.Port, Authoring: options.Authoring,
		Token: options.Token, Workspace: workspace,
	})
}
