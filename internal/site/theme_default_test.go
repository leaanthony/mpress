package site

import (
	"strings"
	"testing"
)

func TestUtilityMenuPanel_CSSInertWhenClosed(t *testing.T) {
	css := defaultThemeCSS

	requiredRules := []string{
		"pointer-events: none",
		"visibility: hidden",
		"pointer-events: auto",
		"visibility: visible",
	}

	for _, rule := range requiredRules {
		if !strings.Contains(css, rule) {
			t.Errorf("missing required CSS rule: %q", rule)
		}
	}
}