package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
		got := formatBytes(tc.input)
		if got != tc.expected {
			t.Errorf("formatBytes(%d) = %s; want %s", tc.input, got, tc.expected)
		}
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()

	if len(m.methods) == 0 {
		t.Fatal("expected methods to be populated")
	}

	if m.urlInput.Value() == "" {
		t.Error("expected default URL to not be empty")
	}

	// Trigger WindowSizeMsg agar width/height terinisialisasi
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := updatedModel.View()

	if !strings.Contains(view, "MARTIS TUI") {
		t.Errorf("expected view to contain header, got: %s", view)
	}
}
