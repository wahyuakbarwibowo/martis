# Repository Guidelines

## Project Structure & Module Organization

Martis is a Go terminal REST client built with Bubble Tea, Bubbles, and Lip Gloss. `main.go` is the entry point; `internal/cli` handles commands. Shared request, response, and collection types live in `internal/domain`; HTTP execution lives in `internal/httpclient`; JSON persistence lives in `internal/repository`. The TUI lives in `internal/ui`, with styling in `internal/ui/styles`. `internal/curlparser` contains cURL parsing code. Tests sit beside package sources as `*_test.go`, with application checks in `main_test.go`. Artwork lives in `assets/`; release configuration lives in `.goreleaser.yaml` and `.github/workflows/release.yml`.

## Build, Test, and Development Commands

Use the Go version declared in `go.mod` (currently `1.27.1`).

- `make run`: launch the TUI with `go run .`.
- `make build`: produce the host-platform `martis` binary with version metadata.
- `make test`: run all tests verbosely with the race detector.
- `make fmt`: format Go packages with `go fmt ./...`.
- `make vet`: run Go static analysis.
- `make tidy`: synchronize module dependencies.
- `make release`: rebuild macOS, Linux, and Windows binaries in `dist/`.

## Coding Style & Naming Conventions

Follow `gofmt` formatting, including tabs for indentation. Use lowercase package names, exported `PascalCase` identifiers, and unexported `camelCase` identifiers. Keep filenames descriptive, such as `collection_repository.go`. Preserve package boundaries: place storage logic in the repository package and HTTP behavior in the client package. Run `make fmt` and `make vet` before submitting code.

## Testing Guidelines

Use Go’s standard `testing` package and `TestXxx` names; parser cases also use names such as `TestParse_SimpleGET`. Prefer table-driven cases for input variations, `httptest.NewServer` for HTTP behavior, and `t.TempDir()` with explicit repository paths for persistence tests. Run focused checks with `go test ./internal/curlparser -run TestParse`. No numeric coverage threshold is configured; cover changed behavior and error paths.

## Commit & Pull Request Guidelines

History uses Conventional Commit prefixes such as `feat:`, `refactor:`, `docs:`, and `ci:`. Write concise, imperative subjects. PRs should explain the change, link relevant issues, and report validation commands and results. Include terminal screenshots for visible TUI changes. Run checks locally: the existing GitHub workflow publishes releases on `v*` tags.

## Configuration & Credentials

Collections and request history persist under `~/martis/`. Environment files live in `~/martis/environments/`. Keep real tokens, private endpoints, and saved user collections out of commits and screenshots.
