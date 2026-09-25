package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"martis/internal/cli"
	"martis/internal/httpclient"
	"martis/internal/repository"
	"martis/internal/ui"
)

// Global version variable di-inject via -ldflags="-X main.version=..."
var version = "dev"

func main() {
	// 1. Process CLI sub-commands (e.g. version, collections, update, help)
	if cli.HandleCLIArgs(version) {
		return
	}

	// 2. Initialize enterprise layers (Dependency Injection)
	repo := repository.NewFileCollectionRepository()
	client := httpclient.NewClient("Martis-TUI-Client/" + version)
	model := ui.NewModel(repo, client)

	// 3. Start Bubble Tea TUI program with mouse & alt screen support
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Martis TUI: %v\n", err)
		os.Exit(1)
	}
}
