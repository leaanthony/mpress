package components

import (
	"strings"
	"testing"
)

func TestCalendarLeavesTodayHighlightToTheBrowser(t *testing.T) {
	calendar := &Calendar{Meta: map[string]string{"month": "2026-08"}}
	output, err := calendar.Render()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, `class="mpress-calendar-cell today"`) {
		t.Fatal("calendar output depends on the date when the static site is built")
	}
	if !strings.Contains(output, `data-date="2026-08-13"`) {
		t.Fatal("calendar did not render the configured month")
	}
}
