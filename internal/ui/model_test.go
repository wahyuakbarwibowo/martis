package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"martis/internal/domain"
)

type testCollectionRepo struct{ collection *domain.Collection }

func (r *testCollectionRepo) Load() (*domain.Collection, error) { return r.collection, nil }
func (r *testCollectionRepo) Save(c *domain.Collection) error   { r.collection = c; return nil }

type testHTTPClient struct{}

func (testHTTPClient) Do(domain.RequestPayload) domain.ResponseResult { return domain.ResponseResult{} }

func TestMouseClickTogglesFolderAtRenderedRow(t *testing.T) {
	col := &domain.Collection{Folders: []domain.Folder{{ID: "f", Name: "Requests", IsExpanded: true, Items: []domain.CollectionItem{{ID: "i", Name: "Get users", Method: "GET", URL: "https://example.test"}}}}}
	m := NewModel(&testCollectionRepo{collection: col}, testHTTPClient{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMsg{X: 4, Y: 4, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = updated.(Model)
	if m.collection.Folders[0].IsExpanded {
		t.Fatal("clicking the displayed folder row should collapse it")
	}
	updated, _ = m.Update(tea.MouseMsg{X: 4, Y: 4, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = updated.(Model)
	if !m.collection.Folders[0].IsExpanded {
		t.Fatal("clicking the displayed folder row should expand it")
	}
	if len(m.sidebarRows) != 2 {
		t.Fatalf("expected expanded folder plus item row, got %d rows", len(m.sidebarRows))
	}
	updated, _ = m.Update(tea.MouseMsg{X: 4, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = updated.(Model)
	if got := m.urlInput.Value(); got != "https://example.test" {
		t.Fatalf("clicking item row loaded URL %q", got)
	}
}
