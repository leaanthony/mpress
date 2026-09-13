package version

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/icons"
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
	if !labelRE.MatchString(label) {
		return fmt.Errorf("invalid version label %q", label)
	}
	cfg, err := config.Load(project)
	if err != nil {
		return err
	}
	src := cfg.OutputPath(project)
	if _, err = os.Stat(filepath.Join(src, "index.html")); err != nil {
		return fmt.Errorf("build output missing; run mpress build first")
	}
	base := cfg.ArtifactsPath(project)
	dest := filepath.Join(base, label)
	if _, err = os.Stat(dest); err == nil && !force {
		return fmt.Errorf("version %s already exists", label)
	}
	tmp := dest + ".tmp"
	_ = os.RemoveAll(tmp)
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.RemoveAll(tmp)
		}
	}()
	if err = copyTree(src, tmp, func(rel string) bool { return strings.HasPrefix(filepath.ToSlash(rel), "versions/") }); err != nil {
		return err
	}
	manifest := Manifest{SchemaVersion: 1, Version: label, CreatedAt: time.Now().UTC().Format(time.RFC3339), Files: map[string]string{}}
	if err = filepath.WalkDir(tmp, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(tmp, path)
		if filepath.Base(path) == "mpress-version.json" {
			return nil
		}
		data, er := os.ReadFile(path)
		if er != nil {
			return er
		}
		sum := sha256.Sum256(data)
		manifest.Files[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	}); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err = os.WriteFile(filepath.Join(tmp, "mpress-version.json"), data, 0o644); err != nil {
		return err
	}
	_ = os.RemoveAll(dest)
	if err = os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	if err = os.Rename(tmp, dest); err != nil {
		return err
	}
	keepTemp = true
	return nil
}
func List(project string) ([]string, error) {
	cfg, err := config.Load(project)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(cfg.ArtifactsPath(project))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && labelRE.MatchString(e.Name()) {
			if info, statErr := os.Stat(filepath.Join(cfg.ArtifactsPath(project), e.Name(), "mpress-version.json")); statErr != nil || !info.Mode().IsRegular() {
				continue
			}
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}
func Verify(project, label string) error {
	cfg, err := config.Load(project)
	if err != nil {
		return err
	}
	root := filepath.Join(cfg.ArtifactsPath(project), label)
	data, err := os.ReadFile(filepath.Join(root, "mpress-version.json"))
	if err != nil {
		return err
	}
	var m Manifest
	if err = json.Unmarshal(data, &m); err != nil {
		return err
	}
	if m.Version != label {
		return fmt.Errorf("manifest version %q does not match %q", m.Version, label)
	}
	for rel, want := range m.Files {
		path, pathErr := manifestFilePath(root, rel)
		if pathErr != nil {
			return pathErr
		}
		data, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			return fmt.Errorf("checksum mismatch: %s", rel)
		}
	}
	if err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "mpress-version.json" {
			return nil
		}
		if _, ok := m.Files[rel]; !ok {
			return fmt.Errorf("unexpected file: %s", rel)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func manifestFilePath(root, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if rel == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid manifest path: %s", rel)
	}
	return filepath.Join(root, clean), nil
}
func Remove(project, label string) error {
	cfg, err := config.Load(project)
	if err != nil {
		return err
	}
	if !labelRE.MatchString(label) {
		return fmt.Errorf("invalid version label")
	}
	return os.RemoveAll(filepath.Join(cfg.ArtifactsPath(project), label))
}
func Mount(project, output string) (int, error) {
	cfg, err := config.Load(project)
	if err != nil || !cfg.Version.Enabled {
		return 0, err
	}
	labels, err := List(project)
	if err != nil {
		return 0, err
	}
	mounted := 0
	for _, label := range labels {
		if err = Verify(project, label); err != nil {
			return mounted, err
		}
		src := filepath.Join(cfg.ArtifactsPath(project), label)
		dest := filepath.Join(output, "versions", label)
		if err = copyTree(src, dest, func(rel string) bool { return filepath.Base(rel) == "mpress-version.json" }); err != nil {
			return mounted, err
		}
		if err = rewriteMountedVersion(dest, label, cfg.Version.Current, labels); err != nil {
			return mounted, err
		}
		mounted++
	}
	data, _ := json.MarshalIndent(map[string]any{"current": cfg.Version.Current, "versions": labels}, "", "  ")
	if err = os.MkdirAll(filepath.Join(output, "versions"), 0o755); err != nil {
		return mounted, err
	}
	err = os.WriteFile(filepath.Join(output, "versions", "versions.json"), data, 0o644)
	return mounted, err
}

func rewriteMountedVersion(root, label, currentLabel string, labels []string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return walkErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rewritten := rewriteRootURLs(string(data), label)
		rewritten = replaceVersionMenu(rewritten, versionRoute(filepath.ToSlash(rel)), label, currentLabel, labels)
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		return os.WriteFile(path, []byte(rewritten), info.Mode().Perm())
	})
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
func copyTree(src, dst string, skip func(string) bool) error {
	return filepath.WalkDir(src, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if skip != nil && skip(rel) {
			if e.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if e.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}
