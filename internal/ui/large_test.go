package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLargeResponseAsksBeforeRendering(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	big := `{"items":[` + strings.Repeat(`{"id":1,"name":"row"},`, 100_000) + `{"id":2}]}` // ~2 MB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, big)
	}))
	defer srv.Close()

	m := newTestModel(t)
	m.methodIndex = 0
	m.urlInput.SetValue(srv.URL)
	m.loading = true
	m, _ = drive(t, m, m.executeRequestCmd())

	view := m.viewport.View()
	if !strings.Contains(view, "Response besar") || strings.Contains(view, `"name": "row"`) {
		t.Fatalf("large body rendered without asking:\n%s", view)
	}

	m.focus = FocusResponse
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if !strings.Contains(m.viewport.View(), `"name": "row"`) {
		t.Fatalf("Enter did not render the body:\n%s", m.viewport.View())
	}
}
