package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"martis/internal/domain"
	"martis/internal/httpclient"
	"martis/internal/repository"
	"martis/internal/ui"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{2048, "2.0 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tc := range tests {
		got := domain.FormatBytes(tc.input)
		if got != tc.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tc.input, got, tc.expected)
		}
	}
}

func TestInitialModel(t *testing.T) {
	repo := repository.NewFileCollectionRepository()
	client := httpclient.NewClient()
	m := ui.NewModel(repo, client)

	// Trigger WindowSizeMsg
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := updatedModel.View()

	if !strings.Contains(view, "martis") {
		t.Errorf("expected view to contain header, got: %s", view)
	}
}
