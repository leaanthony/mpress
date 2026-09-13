package components

import (
	"strings"
	"testing"
)

func TestVideoMinimalSourceUsesNativeDefaults(t *testing.T) {
	rendered, err := (&Video{Meta: map[string]string{"src": "setup.mp4"}}).Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<video controls playsinline preload="metadata" width="1280" height="720">`,
		`<source src="setup.mp4" type="video/mp4">`,
		`<a href="setup.mp4">Open the video file</a>`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered video missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "<figcaption>") {
		t.Fatalf("minimal video rendered an empty caption:\n%s", rendered)
	}
}

func TestVideoResolvesMultipleSourcesAndAccessibleMedia(t *testing.T) {
	rendered, err := (&Video{Meta: map[string]string{
		"base":       "/media/setup",
		"src":        `["setup.webm","setup.mp4"]`,
		"poster":     "poster.webp",
		"captions":   `["captions.en.vtt","captions.fr.vtt"]`,
		"title":      "Setup walkthrough",
		"transcript": "transcript.html",
	}}).Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`poster="/media/setup/poster.webp"`,
		`aria-label="Setup walkthrough"`,
		`<source src="/media/setup/setup.webm" type="video/webm">`,
		`<source src="/media/setup/setup.mp4" type="video/mp4">`,
		`<track kind="captions" src="/media/setup/captions.en.vtt" srclang="en" label="English" default>`,
		`<track kind="captions" src="/media/setup/captions.fr.vtt" srclang="fr" label="fr">`,
		`<figcaption>Setup walkthrough · <a href="/media/setup/transcript.html">Transcript</a></figcaption>`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered video missing %q:\n%s", want, rendered)
		}
	}
	if count := strings.Count(rendered, " default>"); count != 1 {
		t.Fatalf("got %d default caption tracks, want 1:\n%s", count, rendered)
	}
}

func TestVideoAllowsCompactCommaSeparatedInterchangeSources(t *testing.T) {
	rendered, err := (&Video{Meta: map[string]string{
		"src":    "setup.webm, setup.mp4",
		"width":  "960",
		"height": "540",
		"lang":   "cy",
	}}).Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`width="960" height="540"`, `src="setup.webm"`, `src="setup.mp4"`} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered video missing %q:\n%s", want, rendered)
		}
	}
}

func TestVideoRejectsBodiesAndUnsafeURLs(t *testing.T) {
	if err := (&Video{}).Parse("body"); err == nil {
		t.Fatal("video body was accepted")
	}
	for name, metadata := range map[string]map[string]string{
		"source": {"src": "javascript:alert(1)"},
		"base":   {"base": "javascript:alert(1)", "src": "setup.mp4"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := (&Video{Meta: metadata}).Render(); err == nil {
				t.Fatal("unsafe video URL was accepted")
			}
		})
	}
}
