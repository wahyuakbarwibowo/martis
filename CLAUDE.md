# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

See `AGENTS.md` for project layout, coding style, testing conventions, and commit rules. This file only adds what AGENTS.md does not cover.

## Commands

- `make run`: run the TUI. `make build`: build `./martis`; version is injected via `-ldflags -X main.version`. `make desktop-run`: build and open the desktop app.
- `make test`: run `go test -v -race ./...`. For a single test: `go test ./internal/repository -run TestName -v`.
- `make fmt && make vet`: run both before committing.

## Architecture

- `main.go` calls `cli.HandleCLIArgs(version)` first. If it returns true (subcommands such as `import`, `version`, `collections`, `update`, `help`), the process exits without starting the TUI. Otherwise `main.go` wires `repository.NewFileCollectionRepository()` and `httpclient.NewClient(...)` into `ui.NewModel` and runs Bubble Tea with alt screen and mouse cell motion.
- `internal/ui/model.go` is one large Bubble Tea `Model`. It holds focus areas, config tabs, the sidebar tree rows, and the save modal. Requests run asynchronously: `executeRequestCmd` returns a `tea.Cmd`, and `Update` handles the result message. After you change collections, call `rebuildSidebarRows()` so the mouse-clickable sidebar tree stays in sync.
- `internal/requestutil` sits on top of `domain` and `httpclient`. `Prepare` expands `{{vars}}` from environment values. It also provides query-param editing (`Query`/`WithQuery`), response assertions (`Assert`), and `Benchmark`.
- `internal/environment` loads `*.env` files and expands variables.
- `internal/repository/history.go` provides `ConfigDir()` (`~/martis`; `LegacyConfigDir()` is the old `~/.config/martis`) and a shared `WriteJSON` helper. Use them for any new persisted file.
- `internal/importer.File` converts Postman or OpenAPI JSON into a `domain.Collection`; `martis import` appends its folders to the saved collections.
- `internal/curlparser` handles cURL import (`parser.go`) and export (`export.go`).
- `cmd/martis/` is empty. The TUI entry point is the root `main.go`.
- `cmd/martis-desktop/` is a separate Wails desktop binary: `app.go` binds Go methods that call `internal/` packages, and `frontend/` is plain HTML/CSS/JS embedded with `go:embed` (no Node). It needs CGO and `-tags desktop,production`; use `make desktop`. `scripts/package-macos.sh` / `package-linux.sh` wrap the binary into `Martis.app` + `.dmg` or a `.deb` (release CI and `make desktop-app` call them); the app icon source is `cmd/martis-desktop/icon.png`. The TUI must stay buildable with `CGO_ENABLED=0`, so never import Wails outside this directory.

## Notes

- User-facing CLI strings and Makefile messages are in Indonesian. Keep new ones consistent.
