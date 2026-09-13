package components

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"strings"
)

// Tutorial renders an interactive step-by-step tutorial with progress tracking.
// Usage: :::tutorial{title="Getting Started"}
// ### Install M-Press
// Run `go install github.com/leaanthony/mpress@latest`
// ### Create a project
// Run `mpress init my-docs`
// ### Start developing
// Run `mpress dev`
// :::
//
// Each step gets a checkbox for completion tracking (localStorage).
// A progress bar shows overall completion. Steps can be marked complete individually.
type Tutorial struct {
	Meta  map[string]string
	Items []TutorialStep
}

type TutorialStep struct {
	Title   string
	Content string
}

func (t *Tutorial) Parse(content string) error {
	locs := stepHeaderRegex.FindAllStringSubmatchIndex(content, -1)

	if len(locs) == 0 {
		t.Items = append(t.Items, TutorialStep{Title: "Step 1", Content: strings.TrimSpace(content)})
		return nil
	}

	for i, loc := range locs {
		title := content[loc[2]:loc[3]]
		contentStart := loc[1]
		var contentEnd int
		if i+1 < len(locs) {
			contentEnd = locs[i+1][0]
		} else {
			contentEnd = len(content)
		}
		stepContent := strings.TrimSpace(content[contentStart:contentEnd])
		t.Items = append(t.Items, TutorialStep{Title: title, Content: stepContent})
	}
	return nil
}

func (t *Tutorial) Render() (string, error) {
	title := t.Meta["title"]
	if title == "" {
		title = "Tutorial"
	}

	// Generate a stable ID from the title for localStorage key
	tutorialID := generateTutorialID(title)

	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<div class="mpress-tutorial" data-tutorial-id="%s">`, html.EscapeString(tutorialID)))
	b.WriteString("\n")

	// Header with title and progress bar
	b.WriteString(`<div class="mpress-tutorial-header">`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`<h3 class="mpress-tutorial-title">%s</h3>`, html.EscapeString(title)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`<div class="mpress-tutorial-progress"><div class="mpress-tutorial-progress-bar" style="width: 0%%"></div></div>`))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`<span class="mpress-tutorial-count">0/%d completed</span>`, len(t.Items)))
	b.WriteString("\n")
	b.WriteString("</div>\n")

	// Steps
	b.WriteString(`<div class="mpress-tutorial-steps">`)
	b.WriteString("\n")

	for i, step := range t.Items {
		stepID := fmt.Sprintf("%s-step-%d", tutorialID, i)
		b.WriteString(fmt.Sprintf(`<div class="mpress-tutorial-step" data-step="%d">`, i))
		b.WriteString("\n")

		// Checkbox + step number + title
		b.WriteString(fmt.Sprintf(`<div class="mpress-tutorial-step-header">`))
		b.WriteString(fmt.Sprintf(`<label class="mpress-tutorial-check"><input type="checkbox" data-step-id="%s" aria-label="Mark step %d as complete"><span class="mpress-tutorial-checkmark">%s</span></label>`, html.EscapeString(stepID), i+1, lucide("check", 13)))
		b.WriteString(fmt.Sprintf(`<span class="mpress-tutorial-step-num">%d</span>`, i+1))
		b.WriteString(fmt.Sprintf(`<span class="mpress-tutorial-step-title">%s</span>`, html.EscapeString(step.Title)))
		b.WriteString("</div>\n")

		// Content
		if step.Content != "" {
			b.WriteString("<div class=\"mpress-tutorial-step-content\">\n\n")
			b.WriteString(step.Content)
			b.WriteString("\n\n</div>\n")
		}

		b.WriteString("</div>\n")
	}

	b.WriteString("</div>\n")

	// Reset button
	b.WriteString(`<div class="mpress-tutorial-footer">`)
	b.WriteString(`<button class="mpress-tutorial-reset" aria-label="Reset tutorial progress">Reset progress</button>`)
	b.WriteString("</div>\n")

	b.WriteString("</div>\n")

	return b.String(), nil
}

func generateTutorialID(title string) string {
	h := sha256.Sum256([]byte(title))
	return "tutorial-" + hex.EncodeToString(h[:4])
}
