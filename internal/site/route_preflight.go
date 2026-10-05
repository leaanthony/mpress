package site

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/knowledge"
	"github.com/leaanthony/mpress/internal/routes"
	"golang.org/x/text/unicode/norm"
)

type outputEntry struct {
	name  string
	owner string
	dir   bool
	page  bool
}

type outputRegistry map[string]outputEntry

// add includes ancestors so collisions cost path depth rather than scanning
// every prior page. Directory names may be shared; a file may never be a parent.
func (r outputRegistry) add(entry outputEntry) error {
	parent := path.Dir(entry.name)
	if parent != "." {
		if err := r.add(outputEntry{name: parent, owner: entry.owner, dir: true, page: entry.page}); err != nil {
			return err
		}
	}
	key := norm.NFC.String(strings.ToLower(entry.name))
	if previous, exists := r[key]; exists {
		if previous.dir && entry.dir {
			return nil
		}
		// Existing static overrides such as 404.html, robots.txt and _headers
		// remain supported. Only pages must have exclusive output filenames.
		if !previous.dir && !entry.dir && !previous.page && !entry.page && previous.name == entry.name {
			return nil
		}
		return fmt.Errorf("output collision at %q: %s conflicts with %s (%q)", entry.name, entry.owner, previous.owner, previous.name)
	}
	r[key] = entry
	return nil
}

func localizedPageOutput(cfg config.Config, lang, filename string) string {
	if lang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
		return lang + "/" + filename
	}
	return filename
}

func validatePageRoute(page *content.Page) error {
	canonical, err := routes.Normalize(page.URLPath)
	if err != nil {
		return err
	}
	if canonical != page.URLPath || page.OutputPath != path.Join(canonical, "index.html") {
		return fmt.Errorf("%w for %s: URL and output paths disagree", routes.ErrUnsafe, page.SourcePath)
	}
	return routes.Output(page.OutputPath)
}

// generatedOutputFiles is the engine-owned file namespace, including both
// knowledge formats because cleanup removes stale plain and gzip artifacts.
func generatedOutputFiles(cfg config.Config) []string {
	generated := []string{"assets/mpress.css", "assets/mpress.js", "404.html", "_headers", "sitemap.xml", "llms.txt", "robots.txt", "mpress-manifest.json"}
	if cfg.Knowledge.Enabled {
		generated = append(generated, knowledge.Directory+"/"+knowledge.ManifestFile)
		for _, filename := range []string{knowledge.PagesFile, knowledge.ChunksFile, knowledge.IndexFile} {
			generated = append(generated, knowledge.Directory+"/"+filename, knowledge.Directory+"/"+filename+".gz")
		}
	}
	if cfg.Contribution.Enabled {
		generated = append(generated, "contribute.sh", "contribute.ps1", "mpress-contribute.sh", "mpress-contribute.ps1")
	}
	if cfg.Version.Enabled {
		generated = append(generated, "versions/versions.json")
	}
	if cfg.Search.Enabled {
		for _, lang := range cfg.Site.Languages {
			generated = append(generated, searchIndexOutputPath(lang, cfg))
		}
	}
	return generated
}

// preflightPageOutputs checks the complete language namespace, copied trees
// and engine files before removal of the last good site. It trusts neither
// cached output paths nor a filename produced by filepath.Join's cleaning.
func preflightPageOutputs(project string, cfg config.Config, pages map[string][]*content.Page, versionLabels []string) error {
	registry := outputRegistry{}
	for _, filename := range generatedOutputFiles(cfg) {
		if err := registry.add(outputEntry{name: filename, owner: "generated file"}); err != nil {
			return err
		}
	}
	if err := registry.addTree(cfg.StaticPath(project), "", "static file", ""); err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, label := range versionLabels {
		if err := registry.addTree(filepath.Join(cfg.ArtifactsPath(project), label), "versions/"+label, "version "+label, "mpress-version.json"); err != nil {
			return err
		}
	}
	for _, lang := range cfg.Site.Languages {
		for _, page := range pages[lang] {
			if err := validatePageRoute(page); err != nil {
				return err
			}
			entry := outputEntry{name: localizedPageOutput(cfg, lang, page.OutputPath), owner: lang + "/" + page.SourcePath, page: true}
			if err := registry.add(entry); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r outputRegistry) addTree(root, prefix, owner, skipFilename string) error {
	return filepath.WalkDir(root, func(filename string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skipFilename != "" && entry.Name() == skipFilename {
			return nil
		}
		rel, err := filepath.Rel(root, filename)
		if err != nil || rel == "." {
			return err
		}
		return r.add(outputEntry{name: path.Join(prefix, filepath.ToSlash(rel)), owner: owner + " " + rel, dir: entry.IsDir()})
	})
}
