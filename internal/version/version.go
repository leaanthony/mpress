package version

import (
	"errors"
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/leaanthony/mpress/internal/icons"
	"github.com/leaanthony/mpress/internal/routes"
)

type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Version       string            `json:"version"`
	CreatedAt     string            `json:"createdAt"`
	Files         map[string]string `json:"files"`
}

var labelRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

var (
	doubleQuotedRootURLAttributeRE = regexp.MustCompile(`(?i)\b(href|src|action|data-[a-z0-9-]+)="(/[^"]*)"`)
	singleQuotedRootURLAttributeRE = regexp.MustCompile(`(?i)\b(href|src|action|data-[a-z0-9-]+)='(/[^']*)'`)
)

const (
	versionMenuStart = `<!--mpress-version-menu:start-->`
	versionMenuEnd   = `<!--mpress-version-menu:end-->`
)

func Capture(project, label string, force bool) error {
	return capture(project, label, force)
}
func List(project string) ([]string, error) {
	files, _, err := openVersionStore(project)
	if errors.Is(err, errMissingStore) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer files.Close()
	unlock, err := lockVersionRead(files)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return listSnapshots(files)
}

func Verify(project, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	files, cfg, err := openVersionStore(project)
	if err != nil {
		return err
	}
	defer files.Close()
	unlock, err := lockVersionRead(files)
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(cfg.ArtifactsPath(project), label)
	if err := cfg.SafeVersionPath(project, path); err != nil {
		return err
	}
	snapshot, err := files.Sub(label)
	if err != nil {
		return err
	}
	defer snapshot.Close()
	return verifyPinnedSnapshot(snapshot, label)
}

func manifestFilePath(root, rel string) (string, error) {
	if routes.Output(rel) != nil {
		return "", fmt.Errorf("invalid manifest path: %s", rel)
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}
func Remove(project, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	files, cfg, err := openVersionStore(project)
	if errors.Is(err, errMissingStore) {
		return nil
	}
	if err != nil {
		return err
	}
	defer files.Close()
	path := filepath.Join(cfg.ArtifactsPath(project), label)
	if err := cfg.SafeVersionPath(project, path); err != nil {
		return err
	}
	lock, err := files.Lock(storeLockFile)
	if err != nil {
		return err
	}
	defer lock.Close()
	return files.RemoveAll(label)
}
func Mount(project, output string) (int, error) {
	return mount(project, output)
}

func rewriteRootURLs(markup, label string) string {
	prefix := "/versions/" + label + "/"
	rewrite := func(expression *regexp.Regexp, quote string, input string) string {
		return expression.ReplaceAllStringFunc(input, func(match string) string {
			parts := expression.FindStringSubmatch(match)
			path := parts[2]
			if strings.HasPrefix(path, "//") || strings.HasPrefix(path, "/versions/") {
				return match
			}
			return parts[1] + "=" + quote + prefix + strings.TrimPrefix(path, "/") + quote
		})
	}
	markup = rewrite(doubleQuotedRootURLAttributeRE, `"`, markup)
	return rewrite(singleQuotedRootURLAttributeRE, `'`, markup)
}

func versionRoute(rel string) string {
	if rel == "index.html" {
		return "/"
	}
	if strings.HasSuffix(rel, "/index.html") {
		return "/" + strings.TrimSuffix(rel, "index.html")
	}
	return "/" + rel
}

func replaceVersionMenu(markup, route, activeLabel, currentLabel string, labels []string) string {
	start := strings.Index(markup, versionMenuStart)
	end := strings.Index(markup, versionMenuEnd)
	if start < 0 || end < start {
		return markup
	}
	end += len(versionMenuEnd)
	menu := versionMenuMarkup(route, activeLabel, currentLabel, labels)
	return markup[:start] + versionMenuStart + menu + versionMenuEnd + markup[end:]
}

func versionMenuMarkup(route, activeLabel, currentLabel string, labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	if strings.TrimSpace(currentLabel) == "" {
		currentLabel = "Current"
	}
	var out strings.Builder
	out.WriteString(`<div class="header-group utility-select utility-menu version-select"><button class="utility-menu-trigger" type="button" popovertarget="mpress-version-menu" aria-label="Select version" aria-haspopup="menu" aria-expanded="false">`)
	out.WriteString(icons.Lucide("git-branch", 19))
	out.WriteString(`<span class="utility-menu-current">` + html.EscapeString(activeLabel) + `</span>`)
	out.WriteString(icons.Lucide("chevron-down", 13))
	out.WriteString(`</button><menu id="mpress-version-menu" class="utility-menu-panel utility-version-menu" popover data-utility-menu>`)
	writeVersionLink := func(label, url string, current bool) {
		out.WriteString(`<li><a href="` + html.EscapeString(url) + `" role="menuitem"`)
		if current {
			out.WriteString(` aria-current="true"`)
		}
		out.WriteString(`><span class="utility-version-label">` + html.EscapeString(label) + `</span>`)
		if current {
			out.WriteString(`<small>Current documentation</small>`)
		}
		out.WriteString(`</a></li>`)
	}
	writeVersionLink(currentLabel, route, false)
	for _, label := range labels {
		writeVersionLink(label, "/versions/"+label+route, label == activeLabel)
	}
	out.WriteString(`</menu></div>`)
	return out.String()
}
