package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestErrorFooterClearsOnNextKey(t *testing.T) {
	m := &model{lastErr: "database is locked", focus: paneProjects}
	key := tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"})

	updated, _ := m.Update(key)
	got := updated.(*model)
	if got.lastErr != "" {
		t.Fatalf("lastErr = %q, want empty after key press", got.lastErr)
	}
}
