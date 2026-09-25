package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"martis/internal/domain"
	"martis/internal/repository"
)

// Mouse handling hard-codes rows and columns; this keeps the view in step.
func TestViewGeometryMatchesMouseMap(t *testing.T) {
	repo := repository.NewFileCollectionRepository(t.TempDir() + "/c.json")
	for _, width := range []int{120, 150} {
		var model tea.Model = NewModel(repo, nil)
		model, _ = model.Update(tea.WindowSizeMsg{Width: width, Height: 38})
		lines := strings.Split(ansi.Strip(model.View()), "\n")
		for y, want := range map[int]string{2: "│ │ GET    POST", 3: "│ │ URL ", 4: "│ │ Hdr  JSON"} {
			if !strings.Contains(lines[y], want) {
				t.Fatalf("width %d row %d moved: %q", width, y, lines[y])
			}
		}
	}
	model := NewModel(repo, nil)
	if model.methods[(7*4+1-1)/7] != domain.SupportedMethods()[4] {
		t.Fatal("method slot mapping changed")
	}
}
