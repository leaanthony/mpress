// Package mpdcorpus generates deterministic, large MPress Document corpora for
// parser profiling. It is deliberately independent of the parser package.
package mpdcorpus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultPages  = 10_000
	DefaultAssets = 256
)

type Config struct {
	Pages  int
	Assets int
}

type Manifest struct {
	Version     int    `json:"version"`
	Pages       int    `json:"pages"`
	Assets      int    `json:"assets"`
	SourceBytes int64  `json:"sourceBytes"`
	AssetBytes  int64  `json:"assetBytes"`
	SHA256      string `json:"sha256"`
}

// VerifySource validates one generated MPD page. The corpus package accepts a
// callback so it remains independent of any parser implementation.
type VerifySource func(name string, source []byte) error

// Source returns one deterministic page. The ten shapes exercise metadata,
// prose, inline syntax, lists, tables, code, references, and nested components.
func Source(index int) []byte {
	return sourceWithAssets(index, DefaultAssets)
}

func sourceWithAssets(index, assetCount int) []byte {
	var out strings.Builder
	out.Grow(1800)
	asset := index % assetCount
	title := "Generated page " + strconv.Itoa(index+1)
	out.WriteString("---\nschema = 1\ntitle = ")
	writeJSONString(&out, title)
	out.WriteString("\ndescription = ")
	writeJSONString(&out, "A deterministic parser benchmark page with realistic technical documentation.")
	out.WriteString("\nslug = ")
	writeJSONString(&out, fmt.Sprintf("reference/generated/%05d", index+1))
	out.WriteString("\norder = ")
	out.WriteString(strconv.Itoa(index + 1))
	out.WriteString("\ndraft = false\ntags = [\"benchmark\",\"generated\"]\nbenchmark.group = ")
	out.WriteString(strconv.Itoa(index % 100))
	out.WriteString("\n---\n\n# ")
	out.WriteString(title)
	out.WriteString("\n\nMPress parses *strong requirements*, _supporting emphasis_, `inline code`, and @mark[important details] in one forward pass.\nThe next physical line is a deliberate hard break with a :rocket: and [documentation link](/guide/).\n\n@image src=\"/assets/images/image-")
	out.WriteString(fmt.Sprintf("%03d", asset))
	out.WriteString(".svg\" alt=\"Generated architecture diagram\"\n\n")
	switch index % 10 {
	case 0:
		out.WriteString("@steps\n@step title=\"Inspect the project\"\nCheck `mpress.yaml` before the build.\n@end\n@step title=\"Build the site\"\n@terminal title=\"Production build\" prompt=\"$\" comment=\"#\"\n$ mpress build --strict\nBuilt 511 pages.\n@end\n@end\n@end\n")
	case 1:
		out.WriteString("@table header search filter sort paginate column-separators page-size=10\n| Package | Platform | Duration |\n| Core | Linux | `18 ms` |\n| Core | macOS | `21 ms` |\n| CLI | Windows | `24 ms` |\n@end\n")
	case 2:
		out.WriteString("@tabs\n@tab label=\"Go\"\n```go\nsite, err := mpress.Build(config)\nif err != nil { return err }\n```\n@end\n@tab label=\"Shell\"\n```sh\nmpress build --strict\n```\n@end\n@end\n")
	case 3:
		out.WriteString("- Install the binary.\n- Create the documentation.\n  - Add the first page.\n  - Preview every theme.\n- [x] Run the checks.\n- [ ] Publish the output.\n\n> Documentation is part of the product.\n>\n> > Stable source ranges make precise edits possible.\n")
	case 4:
		out.WriteString("@note type=\"important\" title=\"Build requirement\"\nThe output must remain deterministic.\n\n@details title=\"Why this matters\"\nReproducible output makes reviews and deployments safer.\n@end\n@end\n")
	case 5:
		out.WriteString("@api method=\"POST\" path=\"/v1/builds\"\nCreates one static documentation build.\n\n| Field | Type |\n| ref | string |\n| strict | Boolean |\n@end\n")
	case 6:
		out.WriteString("@section variant=\"hero\"\n@columns variant=\"hero\"\n@column\n@headline\nFast documents.\nStable output.\n@end\n@end\n@column\n@callout title=\"Native parser\" icon=\"zap\"\nNo runtime is required.\n@end\n@end\n@end\n@end\n")
	case 7:
		out.WriteString("@filetree\ndocs/\n  index.mpd  Home page\n  guide/\n    install.mpd  Installation guide\nassets/\n  logo.svg  Project mark\n@end\n\n@include src=\"shared/prerequisites.mpd\"\n")
	case 8:
		out.WriteString("Automatic destinations include <https://m-press.me> and <docs@example.com>.\nA reference link points to [Go downloads][go-download].[^ranges]\n\n@link id=\"go-download\" destination=\"https://go.dev/dl/\"\n\n@footnote id=\"ranges\"\nEvery node retains its exact byte range.\n@end\n")
	case 9:
		out.WriteString("@pricing cols=\"2\"\n@plan\n## Starter\n$0\n[Start](/start/)\n- One project\n@end\n@plan recommended\n## Team\n$15\n[Upgrade](/team/)\n- Shared projects\n@end\n@end\n")
	}
	out.WriteString("\n## Continue\n\nThis page is @metadata[title]. The generated order is @metadata[order].\n")
	return []byte(out.String())
}

// Generate writes pages, referenced image assets, shared includes, and a
// deterministic manifest below root.
func Generate(root string, config Config) (Manifest, error) {
	if config.Pages <= 0 {
		config.Pages = DefaultPages
	}
	if config.Assets <= 0 {
		config.Assets = DefaultAssets
	}
	pagesRoot := filepath.Join(root, "pages")
	assetsRoot := filepath.Join(root, "assets", "images")
	sharedRoot := filepath.Join(root, "shared")
	for _, directory := range []string{pagesRoot, assetsRoot, sharedRoot} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return Manifest{}, err
		}
	}
	hash := sha256.New()
	manifest := Manifest{Version: 1, Pages: config.Pages, Assets: config.Assets}
	for index := 0; index < config.Assets; index++ {
		asset := []byte(svgAsset(index))
		name := filepath.Join(assetsRoot, fmt.Sprintf("image-%03d.svg", index))
		if err := writeFile(name, asset); err != nil {
			return Manifest{}, err
		}
		manifest.AssetBytes += int64(len(asset))
		writeDigestEntry(hash, filepath.ToSlash(name[len(root)+1:]), asset)
	}
	shared := []byte("# Prerequisites\n\nInstall Go and verify the toolchain before you continue.\n")
	if err := writeFile(filepath.Join(sharedRoot, "prerequisites.mpd"), shared); err != nil {
		return Manifest{}, err
	}
	manifest.AssetBytes += int64(len(shared))
	writeDigestEntry(hash, "shared/prerequisites.mpd", shared)
	for index := 0; index < config.Pages; index++ {
		group := filepath.Join(pagesRoot, fmt.Sprintf("%03d", index/100))
		if err := os.MkdirAll(group, 0o755); err != nil {
			return Manifest{}, err
		}
		page := sourceWithAssets(index, config.Assets)
		name := filepath.Join(group, fmt.Sprintf("page-%05d.mpd", index+1))
		if err := writeFile(name, page); err != nil {
			return Manifest{}, err
		}
		manifest.SourceBytes += int64(len(page))
		writeDigestEntry(hash, filepath.ToSlash(name[len(root)+1:]), page)
	}
	manifest.SHA256 = hex.EncodeToString(hash.Sum(nil))
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	encoded = append(encoded, '\n')
	if err := writeFile(filepath.Join(root, "manifest.json"), encoded); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Verify reads every generated page and asset in canonical generation order,
// recomputes the manifest digest, and optionally validates every MPD page.
func Verify(root string, verifySource VerifySource) (Manifest, error) {
	encoded, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return Manifest{}, fmt.Errorf("read corpus manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode corpus manifest: %w", err)
	}
	if manifest.Version != 1 || manifest.Pages <= 0 || manifest.Assets <= 0 {
		return Manifest{}, fmt.Errorf("invalid corpus manifest: %#v", manifest)
	}
	hash := sha256.New()
	var sourceBytes, assetBytes int64
	for index := 0; index < manifest.Assets; index++ {
		relative := filepath.ToSlash(filepath.Join("assets", "images", fmt.Sprintf("image-%03d.svg", index)))
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return Manifest{}, fmt.Errorf("read corpus asset %s: %w", relative, err)
		}
		assetBytes += int64(len(contents))
		writeDigestEntry(hash, relative, contents)
	}
	sharedName := "shared/prerequisites.mpd"
	shared, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sharedName)))
	if err != nil {
		return Manifest{}, fmt.Errorf("read shared include: %w", err)
	}
	assetBytes += int64(len(shared))
	writeDigestEntry(hash, sharedName, shared)
	for index := 0; index < manifest.Pages; index++ {
		relative := filepath.ToSlash(filepath.Join("pages", fmt.Sprintf("%03d", index/100), fmt.Sprintf("page-%05d.mpd", index+1)))
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return Manifest{}, fmt.Errorf("read corpus page %s: %w", relative, err)
		}
		sourceBytes += int64(len(contents))
		writeDigestEntry(hash, relative, contents)
		if verifySource != nil {
			if err := verifySource(relative, contents); err != nil {
				return Manifest{}, fmt.Errorf("verify corpus page %s: %w", relative, err)
			}
		}
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if sourceBytes != manifest.SourceBytes || assetBytes != manifest.AssetBytes || digest != manifest.SHA256 {
		return Manifest{}, fmt.Errorf("corpus manifest mismatch: source=%d/%d assets=%d/%d sha256=%s/%s", sourceBytes, manifest.SourceBytes, assetBytes, manifest.AssetBytes, digest, manifest.SHA256)
	}
	return manifest, nil
}

func writeJSONString(out *strings.Builder, value string) {
	encoded, _ := json.Marshal(value)
	out.Write(encoded)
}

func svgAsset(index int) string {
	hue := (index * 47) % 360
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 480" role="img" aria-label="Generated benchmark diagram %d"><rect width="960" height="480" rx="24" fill="hsl(%d 24%% 14%%)"/><rect x="72" y="72" width="816" height="336" rx="16" fill="hsl(%d 18%% 22%%)" stroke="hsl(%d 70%% 62%%)"/><path d="M144 160h336M144 224h560M144 288h448" stroke="hsl(0 0%% 88%%)" stroke-width="20" stroke-linecap="round"/><circle cx="768" cy="168" r="48" fill="hsl(%d 70%% 62%%)"/></svg>`+"\n", index, hue, hue, hue, hue)
}

func writeFile(name string, contents []byte) error { return os.WriteFile(name, contents, 0o644) }

func writeDigestEntry(hash io.Writer, name string, contents []byte) {
	_, _ = io.WriteString(hash, name)
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(contents)
	_, _ = hash.Write([]byte{0})
}
