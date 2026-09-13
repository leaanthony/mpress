package mpdcorpus

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSourceIsDeterministicAndVaried(t *testing.T) {
	first := Source(42)
	if got := Source(42); !reflect.DeepEqual(got, first) {
		t.Fatal("same page index produced different bytes")
	}
	if got := Source(43); reflect.DeepEqual(got, first) {
		t.Fatal("different page indexes produced identical bytes")
	}
}

func TestGenerateCorpus(t *testing.T) {
	firstRoot := filepath.Join(t.TempDir(), "first")
	secondRoot := filepath.Join(t.TempDir(), "second")
	config := Config{Pages: 120, Assets: 12}
	first, err := Generate(firstRoot, config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(secondRoot, config)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("manifests differ:\nfirst  %#v\nsecond %#v", first, second)
	}
	if first.Pages != 120 || first.Assets != 12 || first.SourceBytes == 0 || first.AssetBytes == 0 || len(first.SHA256) != 64 {
		t.Fatalf("invalid manifest: %#v", first)
	}
	for _, name := range []string{"manifest.json", "pages/000/page-00001.mpd", "pages/001/page-00120.mpd", "assets/images/image-000.svg", "shared/prerequisites.mpd"} {
		if info, err := os.Stat(filepath.Join(firstRoot, name)); err != nil || info.Size() == 0 {
			t.Errorf("%s was not generated: %v", name, err)
		}
	}
	verified, err := Verify(firstRoot, func(name string, source []byte) error {
		if len(source) == 0 {
			t.Fatalf("%s is empty", name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if verified != first {
		t.Fatalf("verified manifest differs: %#v != %#v", verified, first)
	}
}
