package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/wailsapp/wails/v2/pkg/runtime"

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

	ctx    context.Context
	mu     sync.Mutex
	cancel context.CancelFunc // stops the request in flight (used for SSE)

	lastBody string                   // raw bytes of the latest response, for downloads
	lastFile requestutil.ResponseFile // download name when the response is a file
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
	Stream     bool              `json:"stream"`
	// File is set when the response is a document, archive, or media file.
	File *requestutil.ResponseFile `json:"file"`
	// Binary means Body was withheld: raw bytes cannot cross to the page intact.
	Binary bool `json:"binary"`
}

func NewApp(version string) *App {
	return &App{
		repo:     repository.NewFileCollectionRepository(),
		client:   httpclient.NewClient("Martis-Desktop/" + version),
		captured: map[string]map[string]string{},
	}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// StopStream cancels the request in flight, ending an open event stream.
func (a *App) StopStream() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
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
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()
	defer cancel()
	// Each SSE event is pushed to the page as it arrives.
	r := httpclient.Stream(ctx, a.client, prepared, func(event string) { runtime.EventsEmit(a.ctx, "sse", event) })
	out := Response{Stream: httpclient.IsEventStream(r.Headers), Status: r.StatusCode, StatusText: http.StatusText(r.StatusCode), DurationMs: r.Duration.Milliseconds(), Size: len(r.Body), Body: r.Body, Headers: map[string]string{}}
	if r.Err != nil {
		out.Error = r.Err.Error()
		return out
	}
	a.mu.Lock()
	a.lastBody, a.lastFile = r.Body, requestutil.ResponseFile{Name: "response.txt"}
	if file, ok := requestutil.DetectFile(r.Headers, r.Body); ok {
		a.lastFile, out.File = file, &file
		out.Binary = !utf8.ValidString(r.Body)
		if out.Binary {
			out.Body = ""
		}
	} else if strings.Contains(r.Headers.Get("Content-Type"), "json") {
		a.lastFile.Name = "response.json"
	}
	a.mu.Unlock()
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

// SaveResponse asks where to save the latest response and writes its raw bytes.
// It returns the saved path, or "" when the user cancels.
func (a *App) SaveResponse() (string, error) {
	a.mu.Lock()
	body, name := a.lastBody, a.lastFile.Name
	a.mu.Unlock()
	downloads := ""
	if home, err := os.UserHomeDir(); err == nil {
		downloads = filepath.Join(home, "Downloads")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{DefaultDirectory: downloads, DefaultFilename: name, Title: "Download response"})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, []byte(body), 0600)
}

// JSONPath filters a response body, mirroring the TUI's json.path search.
func (a *App) JSONPath(body, path string) (string, bool) { return requestutil.JSONPath(body, path) }

func (a *App) Diff(old, new string) string { return requestutil.Diff(old, new) }

func envDir() string { return filepath.Join(repository.ConfigDir(), "environments") }
