package contribute

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/translate"
	"golang.org/x/net/html"
)

const maxPageSize = 2 << 20

// Metadata is the public, non-secret handoff embedded in an M-Press page.
type Metadata struct {
	SiteURL    string
	Repository string
	Branch     string
	Source     string
	Route      string
	Guide      string
}

// Resolve accepts either a generated documentation page or a Git repository.
// A repository target lets a generated installer avoid sending the reader's
// current page URL through the shell command.
func Resolve(ctx context.Context, client *http.Client, target, branchOverride string) (Metadata, error) {
	target = strings.TrimSpace(target)
	branchOverride = strings.TrimSpace(branchOverride)
	if isRepositoryTarget(target) {
		if branchOverride == "" {
			branchOverride = "main"
		}
		return Metadata{Repository: target, Branch: branchOverride, Route: "/"}, nil
	}
	metadata, err := Discover(ctx, client, target)
	if err != nil {
		return Metadata{}, err
	}
	if branchOverride != "" {
		metadata.Branch = branchOverride
	}
	return metadata, nil
}

func isRepositoryTarget(target string) bool {
	if strings.HasPrefix(target, "git@") {
		return true
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return false
	}
	if parsed.Scheme == "ssh" || parsed.Scheme == "git" {
		return parsed.Host != "" && strings.Trim(parsed.Path, "/") != ""
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && strings.HasSuffix(strings.TrimSuffix(parsed.Path, "/"), ".git")
}

// Discover reads contribution metadata from a generated M-Press page.
func Discover(ctx context.Context, client *http.Client, rawURL string) (Metadata, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return Metadata{}, errors.New("contribution URL must be an http or https page")
	}
	parsed.Fragment = ""
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Metadata{}, err
	}
	request.Header.Set("User-Agent", "M-Press contributor setup")
	response, err := client.Do(request)
	if err != nil {
		return Metadata{}, fmt.Errorf("open contribution page: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Metadata{}, fmt.Errorf("contribution page returned %s", response.Status)
	}
	document, err := html.Parse(io.LimitReader(response.Body, maxPageSize))
	if err != nil {
		return Metadata{}, fmt.Errorf("read contribution page: %w", err)
	}
	values := map[string]string{}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "meta" {
			name, content := "", ""
			for _, attribute := range node.Attr {
				switch strings.ToLower(attribute.Key) {
				case "name":
					name = strings.ToLower(strings.TrimSpace(attribute.Val))
				case "content":
					content = strings.TrimSpace(attribute.Val)
				}
			}
			if strings.HasPrefix(name, "mpress:") {
				values[name] = content
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	repository := values["mpress:repository"]
	if repository == "" {
		return Metadata{}, errors.New("this page does not advertise an M-Press contribution repository")
	}
	branch := values["mpress:branch"]
	if branch == "" {
		branch = "main"
	}
	route := values["mpress:route"]
	if route == "" {
		route = parsed.EscapedPath()
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	return Metadata{SiteURL: parsed.String(), Repository: repository, Branch: branch, Source: values["mpress:source"], Route: route, Guide: values["mpress:guide"]}, nil
}

// DefaultCheckout returns a visible, stable location that can be reused the
// next time the same project is opened for contribution.
func DefaultCheckout(repository string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	clean := strings.TrimSuffix(strings.TrimSpace(repository), ".git")
	clean = strings.TrimSuffix(clean, "/")
	clean = strings.ReplaceAll(clean, ":", "/")
	parts := strings.FieldsFunc(clean, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return "", errors.New("could not derive a checkout name from the repository URL")
	}
	name := parts[len(parts)-1]
	if len(parts) > 1 {
		name = parts[len(parts)-2] + "-" + name
	}
	name = safeName(name)
	if name == "" {
		return "", errors.New("could not derive a checkout name from the repository URL")
	}
	return filepath.Join(home, "mpress-contributions", name), nil
}

// Prepare clones a project or safely reuses its existing contribution checkout.
func Prepare(ctx context.Context, metadata Metadata, destination string, now time.Time) (root, branch string, reused bool, err error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		destination, err = DefaultCheckout(metadata.Repository)
		if err != nil {
			return "", "", false, err
		}
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return "", "", false, err
	}
	if info, statErr := os.Stat(abs); statErr == nil && info.IsDir() {
		entries, readErr := os.ReadDir(abs)
		if readErr != nil {
			return "", "", false, readErr
		}
		if len(entries) > 0 {
			if _, loadErr := config.Load(abs); loadErr != nil {
				return "", "", false, fmt.Errorf("existing checkout is not an M-Press project: %w", loadErr)
			}
			origin := strings.TrimSpace(gitOutput(abs, "remote", "get-url", "origin"))
			if !sameRepository(origin, metadata.Repository) {
				return "", "", false, fmt.Errorf("existing checkout belongs to %s; choose another directory with --checkout", origin)
			}
			current := strings.TrimSpace(gitOutput(abs, "branch", "--show-current"))
			if strings.HasPrefix(current, "contribute/") {
				return abs, current, true, nil
			}
			if strings.TrimSpace(gitOutput(abs, "status", "--porcelain")) != "" {
				return "", "", false, errors.New("existing checkout has uncommitted changes; commit them or choose another directory with --checkout")
			}
			reused = true
			root = abs
		} else {
			root, err = translate.CloneProject(ctx, metadata.Repository, metadata.Branch, abs)
		}
	} else if errors.Is(statErr, os.ErrNotExist) {
		root, err = translate.CloneProject(ctx, metadata.Repository, metadata.Branch, abs)
	} else if statErr != nil {
		return "", "", false, statErr
	} else {
		return "", "", false, fmt.Errorf("checkout destination %s is not a directory", abs)
	}
	if err != nil {
		return "", "", false, fmt.Errorf("could not clone the contribution repository: %w; for a private repository, sign in with the GitHub CLI or configure Git credentials, then try again", err)
	}
	branch = "contribute/" + now.UTC().Format("20060102-150405")
	if err := translate.PrepareWorkBranch(ctx, root, branch); err != nil {
		return "", "", false, err
	}
	return root, branch, reused, nil
}

// DetectGuide returns a safe project-relative contributor guide. An explicit
// value wins; otherwise common repository conventions are checked in order.
func DetectGuide(root, configured string) string {
	configured = filepath.ToSlash(filepath.Clean(strings.TrimSpace(configured)))
	if configured != "." && configured != "" && !strings.HasPrefix(configured, "../") && !filepath.IsAbs(configured) {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(configured))); err == nil && !info.IsDir() {
			return configured
		}
	}
	for _, candidate := range []string{"CONTRIBUTING.md", ".github/CONTRIBUTING.md", "docs/contributing.md"} {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(candidate))); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func gitOutput(project string, args ...string) string {
	commandArgs := append([]string{"-C", project}, args...)
	output, _ := exec.Command("git", commandArgs...).Output()
	return string(output)
}

func sameRepository(left, right string) bool {
	normalise := func(value string) string {
		value = strings.ToLower(strings.TrimSpace(value))
		value = strings.TrimSuffix(value, ".git")
		value = strings.TrimSuffix(value, "/")
		return value
	}
	return normalise(left) != "" && normalise(left) == normalise(right)
}

func safeName(value string) string {
	var out strings.Builder
	for _, character := range strings.ToLower(value) {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			out.WriteRune(character)
		case character == '-', character == '_', character == '.':
			out.WriteRune(character)
		default:
			out.WriteByte('-')
		}
	}
	return strings.Trim(out.String(), "-.")
}
