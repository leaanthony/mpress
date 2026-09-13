package dev

import (
	"strings"
	"testing"
)

func TestWorkspaceNavigationClearsOnlyAfterUnsavedCheck(t *testing.T) {
	script := devbarJS
	guardedNavigation := "if (!canLeaveWorkspace()) return;\n      resetWorkspaceChangeGuard();\n      const tool = link.dataset.workspaceCommand;"
	if !strings.Contains(script, guardedNavigation) {
		t.Fatal("workspace navigation must check unsaved settings before clearing the guard")
	}
	connectedSettings := "Boolean(q('#config-complete-form')?.isConnected) && settingsState() !== initialSettingsState"
	if !strings.Contains(script, connectedSettings) {
		t.Fatal("a removed settings form must not be reported as unsaved")
	}
}
