package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestFileResponseOffersDownloadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	downloads := home + "/Downloads"
	if err := os.Mkdir(downloads, 0700); err != nil {
		t.Fatal(err)
	}
	pdf := "%PDF-1.7\n\xff\xd8\x00binary\x01"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="invoice.pdf"`)
		fmt.Fprint(w, pdf)
	}))
	defer srv.Close()

	m := newTestModel(t)
	m.methodIndex = 0
	m.urlInput.SetValue(srv.URL)
	m.loading = true
	m, _ = drive(t, m, m.executeRequestCmd())
	if view := m.viewport.View(); !strings.Contains(view, "File PDF · invoice.pdf") || strings.Contains(view, "binary") {
		t.Fatalf("file not summarised:\n%s", view)
	}

	m.focus = FocusResponse
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if strings.Contains(next.(Model).viewport.View(), "binary") {
		t.Fatal("Enter rendered binary content")
	}

	for i := 0; i < 2; i++ { // second download must not overwrite the first
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
		m = next.(Model)
	}
	for _, name := range []string{"invoice.pdf", "invoice (1).pdf"} {
		got, err := os.ReadFile(downloads + "/" + name)
		if err != nil || string(got) != pdf {
			t.Fatalf("%s: err=%v bytes-equal=%v", name, err, string(got) == pdf)
		}
	}
}
