package media

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// Finding is one video compatibility problem on one page.
type Finding struct {
	Page    string // output-relative HTML path
	Source  string // URL of the first local source
	File    string // that source on disk
	Message string
}

func (f Finding) String() string { return f.Page + ": " + f.Message }

// Transcoder produces a universally playable H.264 MP4 from src at dst.
// M-Press ships no implementation yet; Audit only reports what is missing.
type Transcoder interface {
	ToH264(src, dst string) error
}

// AuditHTML checks every <video> in one rendered page. Sources that are not
// local files in output (remote URLs, missing files) are skipped: link
// checking owns those. A video passes when any source plays everywhere.
func AuditHTML(output, relative string, data []byte) []Finding {
	if !bytes.Contains(data, []byte("<video")) {
		return nil
	}
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	var findings []Finding
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "video" {
			if f, ok := auditVideo(output, relative, n); ok {
				findings = append(findings, f)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return findings
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func auditVideo(output, relative string, video *html.Node) (Finding, bool) {
	var sources []string
	if src := attr(video, "src"); src != "" {
		sources = append(sources, src)
	}
	for c := video.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "source" && attr(c, "src") != "" {
			sources = append(sources, attr(c, "src"))
		}
	}
	if len(sources) == 0 {
		return Finding{}, false
	}
	var notes []string
	var firstFile string
	for _, source := range sources {
		file, ok := localFile(output, relative, source)
		if !ok {
			return Finding{}, false // remote or unresolvable: cannot judge
		}
		if firstFile == "" {
			firstFile = file
		}
		info, err := ProbeFile(file)
		if err != nil {
			return Finding{}, false
		}
		play := info.Playback()
		if play.Universal {
			return Finding{}, false
		}
		notes = append(notes, fmt.Sprintf("%s (%s)", path.Base(source), play.Note))
	}
	return Finding{
		Page:   relative,
		Source: sources[0],
		File:   firstFile,
		Message: "video has no H.264 MP4 source, so it will not play on some devices (for example iPad Safari): " +
			strings.Join(notes, "; ") + ".",
	}, true
}

func localFile(output, relative, source string) (string, bool) {
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "", false
	}
	p := u.Path
	if !strings.HasPrefix(p, "/") {
		p = path.Join(path.Dir("/"+filepath.ToSlash(relative)), p)
	}
	file := filepath.Join(output, filepath.FromSlash(path.Clean(p)))
	if !filepath.IsLocal(strings.TrimPrefix(filepath.ToSlash(path.Clean(p)), "/")) {
		return "", false
	}
	if _, err := os.Stat(file); err != nil {
		return "", false
	}
	return file, true
}

// AuditTree audits every HTML page under output.
func AuditTree(output string) ([]Finding, error) {
	var findings []Finding
	err := filepath.WalkDir(output, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".html" {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(output, p)
		findings = append(findings, AuditHTML(output, filepath.ToSlash(rel), data)...)
		return nil
	})
	return findings, err
}
