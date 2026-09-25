package ui

import (
	"runtime"
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

func TestMouseClickOpensFormFilePicker(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS uses the native Finder picker")
	}
	m := NewModel(&testCollectionRepo{collection: &domain.Collection{}}, testHTTPClient{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	m.tab = TabBodyForm
	updated, _ = m.Update(tea.MouseMsg{X: 40, Y: 10, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = updated.(Model)
	if m.modal != modalFilePicker || m.formFocusIndex != 1 {
		t.Fatalf("clicking form path should open picker: modal=%v index=%d", m.modal, m.formFocusIndex)
	}
}

func TestFormFilePickerLoadsSelectedFile(t *testing.T) {
	m := NewModel(&testCollectionRepo{collection: &domain.Collection{}}, testHTTPClient{})
	m.tab = TabBodyForm
	m.openFilePicker()
	if m.modal != modalFilePicker {
		t.Fatal("Ctrl+F should open the form-data file picker")
	}
	if len(m.fileCandidates) == 0 {
		t.Skip("repository has no regular files")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.modal != modalNone || m.formFilePath.Value() == "" || m.tab != TabBodyForm {
		t.Fatalf("picker should load selected path: modal=%v path=%q tab=%v", m.modal, m.formFilePath.Value(), m.tab)
	}
}

func TestRenameCollectionRequest(t *testing.T) {
	col := &domain.Collection{Folders: []domain.Folder{{ID: "f", Name: "Requests", IsExpanded: true, Items: []domain.CollectionItem{{ID: "i", Name: "Old", Method: "GET", URL: "https://example.test"}}}}}
	m := NewModel(&testCollectionRepo{collection: col}, testHTTPClient{})
	m.selectedTreeIndex = 1
	m.sidebarRows = []treeRow{{rowType: rowFolder, folderIndex: 0}, {rowType: rowItem, folderIndex: 0, itemIndex: 0}}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(Model)
	if m.modal != modalRename {
		t.Fatal("r should open rename modal")
	}
	m.modalInput.SetValue("New")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.collection.Folders[0].Items[0].Name != "New" {
		t.Fatalf("request was not renamed: %#v", m.collection.Folders[0].Items[0])
	}
}

func TestFormEditorSupportsMultipleFieldsAndFiles(t *testing.T) {
	m := NewModel(&testCollectionRepo{collection: &domain.Collection{}}, testHTTPClient{})
	m.tab = TabBodyForm
	m.formEditor.SetValue("name=martis\nrole=admin\n@avatar=/tmp/avatar.png\n@document=/tmp/doc.pdf")
	p := m.currentPayload()
	if len(p.FormFields) != 2 || len(p.FormFiles) != 1 || p.FormKey != "avatar" || p.FormPath != "/tmp/avatar.png" {
		t.Fatalf("unexpected form payload: %#v", p)
	}
}

func TestSidebarDeletesRequest(t *testing.T) {
	col := &domain.Collection{Folders: []domain.Folder{{ID: "f", Name: "Requests", IsExpanded: true, Items: []domain.CollectionItem{{ID: "i", Name: "Get", Method: "GET", URL: "https://example.test"}}}}}
	m := NewModel(&testCollectionRepo{collection: col}, testHTTPClient{})
	m.selectedTreeIndex = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)
	if len(m.collection.Folders[0].Items) != 0 {
		t.Fatal("sidebar delete should remove selected request")
	}
}
