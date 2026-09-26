package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"martis/internal/repository"
)

func TestF4FormatsRawJSONBody(t *testing.T) {
	m := NewModel(repository.NewFileCollectionRepository(t.TempDir()+"/c.json"), nil)
	m.tab = TabBodyRaw
	m.jsonBody.SetValue(`{"a":1,"b":[true]}`)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF4})
	m = next.(Model)
	if got := m.jsonBody.Value(); got != "{\n  \"a\": 1,\n  \"b\": [\n    true\n  ]\n}" {
		t.Fatalf("body not formatted: %q", got)
	}

	m.jsonBody.SetValue("{broken")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyF4})
	m = next.(Model)
	if m.jsonBody.Value() != "{broken" || m.status != "Body bukan JSON valid" {
		t.Fatalf("invalid body changed or no status: %q / %q", m.jsonBody.Value(), m.status)
	}
}
