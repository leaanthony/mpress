package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/leaanthony/mpress/internal/mpd"
	"github.com/leaanthony/mpress/internal/mpdcorpus"
)

func main() {
	output := flag.String("output", "mpd-corpus-10000", "output directory")
	pages := flag.Int("pages", mpdcorpus.DefaultPages, "number of MPD pages")
	assets := flag.Int("assets", mpdcorpus.DefaultAssets, "number of referenced image assets")
	flag.Parse()
	if err := os.MkdirAll(*output, 0o755); err != nil {
		log.Fatal(err)
	}
	manifest, err := mpdcorpus.Generate(*output, mpdcorpus.Config{Pages: *pages, Assets: *assets})
	if err != nil {
		log.Fatal(err)
	}
	verified, err := mpdcorpus.Verify(*output, func(name string, source []byte) error {
		document := mpd.Parse(name, source)
		for _, diagnostic := range document.Diagnostics {
			if diagnostic.Severity == mpd.SeverityError {
				return fmt.Errorf("%s at %d:%d: %s", diagnostic.Code, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.Message)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("generated %d pages (%d bytes) and %d assets (%d bytes)\nsha256 %s\n", manifest.Pages, manifest.SourceBytes, manifest.Assets, manifest.AssetBytes, manifest.SHA256)
	fmt.Printf("verified %d pages and %d assets\n", verified.Pages, verified.Assets)
}
