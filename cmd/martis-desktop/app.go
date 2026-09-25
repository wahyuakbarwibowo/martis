package main

import (
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"martis/internal/domain"
	"martis/internal/environment"
	"martis/internal/httpclient"
	"martis/internal/repository"
	"martis/internal/requestutil"
)

// App is bound to the frontend; every exported method is callable from JS.
type App struct {
	repo   repository.CollectionRepository
	client httpclient.Client
	// captured holds "set name = json.path" values per environment for this session.
	captured map[string]map[string]string
}

type Response struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	DurationMs int64             `json:"durationMs"`
	Size       int               `json:"size"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error"`
	Failures   []string          `json:"failures"`
	Captured   []string          `json:"captured"`
}

func NewApp(version string) *App {
	return &App{
		repo:     repository.NewFileCollectionRepository(),
		client:   httpclient.NewClient("Martis-Desktop/" + version),
		captured: map[string]map[string]string{},
	}
}

func (a *App) Collections() (*domain.Collection, error) { return a.repo.Load() }

func (a *App) SaveCollections(col *domain.Collection) error { return a.repo.Save(col) }

// Environments lists env names (without .env) from the config directory.
func (a *App) Environments() []string {
	files, _ := environment.Files(envDir())
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), ".env"))
	}
	sort.Strings(names)
	return names
}

func (a *App) Send(p domain.RequestPayload, env string) Response {
	vars := map[string]string{}
	if env != "" {
		loaded, err := environment.Load(filepath.Join(envDir(), filepath.Base(env)+".env"))
		if err != nil {
			return Response{Error: err.Error()}
		}
		vars = loaded
	}
	for k, v := range a.captured[env] {
		vars[k] = v
	}
	prepared, err := requestutil.Prepare(p, vars)
	if err != nil {
		return Response{Error: err.Error()}
	}
	r := a.client.Do(prepared)
	out := Response{Status: r.StatusCode, StatusText: http.StatusText(r.StatusCode), DurationMs: r.Duration.Milliseconds(), Size: len(r.Body), Body: r.Body, Headers: map[string]string{}}
	if r.Err != nil {
		out.Error = r.Err.Error()
		return out
	}
	for k, v := range r.Headers {
		out.Headers[k] = strings.Join(v, ", ")
	}
	captured, captureFailures := requestutil.Capture(r, p.Assertions)
	if len(captured) > 0 && a.captured[env] == nil {
		a.captured[env] = map[string]string{}
	}
	for k, v := range captured {
		a.captured[env][k] = v
		out.Captured = append(out.Captured, k)
	}
	sort.Strings(out.Captured)
	out.Failures = append(requestutil.Assert(r, p.Assertions), captureFailures...)
	return out
}

// JSONPath filters a response body, mirroring the TUI's json.path search.
func (a *App) JSONPath(body, path string) (string, bool) { return requestutil.JSONPath(body, path) }

func (a *App) Diff(old, new string) string { return requestutil.Diff(old, new) }

func envDir() string { return filepath.Join(repository.ConfigDir(), "environments") }
