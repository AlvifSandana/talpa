package cmd

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuModelUpdateNavigationAndSelection(t *testing.T) {
	m := newMenuModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := updated.(menuModel)
	if m2.cursor != 1 {
		t.Fatalf("expected cursor to move down, got %d", m2.cursor)
	}

	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := updated.(menuModel)
	if len(m3.selected) == 0 {
		t.Fatalf("expected selected command args")
	}
	if got := strings.Join(m3.selected, " "); got != "analyze" {
		t.Fatalf("unexpected selection: %s", got)
	}
}

func TestMenuModelUpdateQuit(t *testing.T) {
	m := newMenuModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2 := updated.(menuModel)
	if !m2.exit {
		t.Fatalf("expected exit to be true")
	}
}

func TestMenuModelViewContainsTitleAndHint(t *testing.T) {
	view := newMenuModel().View()
	if !strings.Contains(view, "Talpa Interactive") {
		t.Fatalf("expected view title")
	}
	if !strings.Contains(view, "Enter to run") {
		t.Fatalf("expected hint text")
	}
}
