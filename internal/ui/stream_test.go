package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"martis/internal/domain"
	"martis/internal/httpclient"
	"martis/internal/repository"
)

// drive runs a request command to completion, feeding every message back into Update.
func drive(t *testing.T, m Model, cmd tea.Cmd) (Model, []string) {
	t.Helper()
	var statuses []string
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		next, nextCmd := m.Update(msg)
		m = next.(Model)
		statuses = append(statuses, m.status)
		if _, done := msg.(domain.ResponseResult); done {
			break
		}
		cmd = nextCmd
	}
	return m, statuses
}

func newTestModel(t *testing.T) Model {
	m := NewModel(repository.NewFileCollectionRepository(t.TempDir()+"/c.json"), httpclient.NewClient())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 38})
	return next.(Model)
}

func TestSSEEventsStreamIntoTheViewer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 1; i <= 3; i++ {
			fmt.Fprintf(w, "data: %d\n\n", i)
			w.(http.Flusher).Flush()
		}
	}))
	defer srv.Close()

	m := newTestModel(t)
	m.methodIndex = 0 // GET
	m.urlInput.SetValue(srv.URL)
	m.loading = true
	m, statuses := drive(t, m, m.executeRequestCmd())
	if !strings.Contains(strings.Join(statuses, "|"), "Streaming · 3 events") {
		t.Fatalf("no live stream status: %q", statuses)
	}
	if m.loading || !strings.Contains(m.responseBody, "data: 3") {
		t.Fatalf("stream did not finish: loading=%v body=%q", m.loading, m.responseBody)
	}
}

func TestGraphQLTabSendsQueryAndVariables(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var got map[string]any
	var contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		fmt.Fprint(w, `{"data":{"ok":true}}`)
	}))
	defer srv.Close()

	m := newTestModel(t)
	m.methodIndex = 1 // POST
	m.urlInput.SetValue(srv.URL)
	m.tab = TabGraphQL
	m.configEditor.SetValue("query { user(id: $id) { name } }\n" + gqlSeparator + "\n{\"id\": \"7\"}")
	m.loading = true
	m, _ = drive(t, m, m.executeRequestCmd())

	vars, _ := got["variables"].(map[string]any)
	if got["query"] != "query { user(id: $id) { name } }" || vars["id"] != "7" {
		t.Fatalf("server got %v", got)
	}
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("content type %q", contentType)
	}
	if item := m.payloadToItem(m.lastPayload); item.BodyType != "graphql" || item.Variables != `{"id": "7"}` {
		t.Fatalf("history item lost GraphQL fields: %+v", item)
	}
}
