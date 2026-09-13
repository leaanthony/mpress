package site

import (
	"errors"
	"testing"
)

func TestRenderPagesPropagatesPreparationError(t *testing.T) {
	want := errors.New("prepare page")
	_, _, err := renderPages(t.TempDir(), 1, false, false, func(int) (string, templateData, error) {
		return "", templateData{}, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("renderPages returned %v, want %v", err, want)
	}
}
